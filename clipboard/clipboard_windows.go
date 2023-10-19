//go:build windows
// +build windows

package clipboard

import (
  "clipboard-remote/utils"
  "context"
  "fmt"
  "os"
  "path/filepath"
  "reflect"
  "runtime"
  "sync"
  "syscall"
  "time"
  "unsafe"

  log "github.com/sirupsen/logrus"
)

const (
  cfUNICODETEXT = 13 //text
  cfHDROP       = 15 //files
  gmemMoveable  = 0x0002
  gMemGHND      = 0x0042
)

var (
  user32                     = syscall.MustLoadDLL("user32")
  getPriorityClipboardFormat = user32.MustFindProc("GetPriorityClipboardFormat")
  getClipboardSequenceNumber = user32.MustFindProc("GetClipboardSequenceNumber")

  openClipboard    = user32.MustFindProc("OpenClipboard")
  closeClipboard   = user32.MustFindProc("CloseClipboard")
  emptyClipboard   = user32.MustFindProc("EmptyClipboard")
  getClipboardData = user32.MustFindProc("GetClipboardData")
  setClipboardData = user32.MustFindProc("SetClipboardData")

  kernel32     = syscall.NewLazyDLL("kernel32")
  globalAlloc  = kernel32.NewProc("GlobalAlloc")
  globalFree   = kernel32.NewProc("GlobalFree")
  globalLock   = kernel32.NewProc("GlobalLock")
  globalUnlock = kernel32.NewProc("GlobalUnlock")
  moveMemory   = kernel32.NewProc("RtlMoveMemory")
)

var (
  countLock      sync.Mutex
  clipboardCount uintptr
)

// waitOpenClipboard opens the clipboard, waiting for up to a second to do so.
func waitOpenClipboard() error {
  started := time.Now()
  limit := started.Add(time.Second)
  var r uintptr
  var err error
  for time.Now().Before(limit) {
    r, _, err = openClipboard.Call(0)
    if r != 0 {
      return nil
    }
    time.Sleep(time.Millisecond)
  }
  return err
}

// use this func need lock the memory first
func convertBufToStr(p uintptr) (int, string) {
  // Count until NUL terminator
  n := 0
  for ptr := unsafe.Pointer(p); *(*uint16)(ptr) != 0; n++ {
    ptr = unsafe.Pointer(uintptr(ptr) + unsafe.Sizeof(uint16(0)))
  }

  var s []uint16
  h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
  h.Data = p
  h.Len = n
  h.Cap = n

  return n, syscall.UTF16ToString(s)
}

func read() ([]byte, error) {
  // LockOSThread ensure that the whole method will keep executing on the same thread from begin to end (it actually locks the goroutine thread attribution).
  // Otherwise if the goroutine switch thread during execution (which is a common practice), the OpenClipboard and CloseClipboard will happen on two different threads, and it will result in a clipboard deadlock.
  runtime.LockOSThread()
  defer runtime.UnlockOSThread()

  availableFormats := [3]uint{cfHDROP, cfUNICODETEXT, 0}

  format, _, err := getPriorityClipboardFormat.Call(uintptr(unsafe.Pointer(&availableFormats[0])), uintptr(len(availableFormats)+1))
  if int(format) == -1 {
    log.Errorln("Clipborad data format is not available.", err)
    return nil, err
  }

  err = waitOpenClipboard()
  if err != nil {
    return nil, err
  }
  defer closeClipboard.Call()

  switch format {
  case cfHDROP:
    return readFilePath()
  case cfUNICODETEXT:
    return readText()
  default:
    return nil, fmt.Errorf("unsupported clipboard format %d", format)
  }
}

// write data to clipboard
func write(buf []byte) (<-chan struct{}, error) {
  errch := make(chan error)
  changed := make(chan struct{}, 1)
  go func() {
    // make sure GetClipboardSequenceNumber happens with
    // OpenClipboard on the same thread.
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()

    err := waitOpenClipboard()
    if err != nil {
      log.Errorln("Failed to open clipboard:", err)
      return
    }

    clipInfo, err := utils.DecodeToStruct(buf)
    if err != nil {
      log.Errorln("Failed to decode struct:", err)
      closeClipboard.Call()
      return
    }

    // exclusive with watch
    countLock.Lock()
    defer countLock.Unlock()

    switch clipInfo.Type {
    case utils.CLIP_PATH:
      err := writeFilePath(clipInfo.Name, clipInfo.Buff)
      if err != nil {
        errch <- err
        closeClipboard.Call()
        return
      }
    case utils.CLIP_TEXT:
      fallthrough
    default:
      err := writeText(clipInfo.Buff)
      if err != nil {
        errch <- err
        closeClipboard.Call()
        return
      }
    }
    // Close the clipboard otherwise other applications cannot
    // paste the data.
    closeClipboard.Call()

    //clipboardCount, _, _ = getClipboardSequenceNumber.Call()
    errch <- nil
    for {
      time.Sleep(time.Second)
      cur, _, _ := getClipboardSequenceNumber.Call()
      log.Debugf("Write succeed: last count %d, new count: %d.", clipboardCount, cur)
      if cur != clipboardCount {
        changed <- struct{}{}
        close(changed)
        clipboardCount = cur
        return
      }
    }
  }()

  err := <-errch
  if err != nil {
    return nil, err
  }
  return changed, nil
}

// readText reads the clipboard and returns the text data if presents.
// The caller is responsible for opening/closing the clipboard before
// calling this function.
func readText() ([]byte, error) {
  hMem, _, err := getClipboardData.Call(cfUNICODETEXT)
  if hMem == 0 {
    return nil, err
  }
  p, _, err := globalLock.Call(hMem)
  if p == 0 {
    return nil, err
  }
  defer globalUnlock.Call(hMem)

  _, content := convertBufToStr(p)

  buff := utils.ClipBoardBuff{
    Type: utils.CLIP_TEXT,
    Name: "",
    Buff: utils.StringToBytes(content),
  }

  return utils.EncodeToBytes(buff)
}

// writeText writes given data to the clipboard. It is the caller's
// responsibility for opening/closing the clipboard before calling
// this function.
func writeText(buf []byte) error {
  r, _, err := emptyClipboard.Call()
  if r == 0 {
    log.Errorln("Failed to clear clipboard:", err)
    return err
  }

  if len(buf) == 0 {
    return nil
  }

  s, err := syscall.UTF16FromString(string(buf))
  if err != nil {
    log.Errorln("Failed to convert given string:", err)
    return err
  }

  textLen := len(s) * int(unsafe.Sizeof(s[0]))

  hMem, _, err := globalAlloc.Call(gmemMoveable, uintptr(textLen))
  if hMem == 0 {
    log.Errorln("Failed to alloc global memory:", err)
    return err
  }

  pMem, _, err := globalLock.Call(hMem)
  if pMem == 0 {
    log.Errorln("Failed to lock global memory:", err)
    return err
  }
  defer globalUnlock.Call(hMem)

  // no return value
  moveMemory.Call(pMem, uintptr(unsafe.Pointer(&s[0])), uintptr(textLen))

  handle, _, err := setClipboardData.Call(cfUNICODETEXT, hMem)
  if handle == 0 {
    globalFree.Call(hMem)
    log.Errorln("Failed to set text to clipboard:", err)
    return err
  }

  return nil
}

func fileRead(filePath string) ([]byte, error) {
  buffer, err := os.ReadFile(filePath)
  if err != nil {
    log.Errorln("Failed to read file:", filePath)
    return nil, err
  }

  buff := utils.ClipBoardBuff{
    Type: utils.CLIP_PATH,
    Name: filepath.Base(filePath),
    Buff: buffer,
  }

  return utils.EncodeToBytes(buff)
}

func isDirExist(path string) bool {
  s, err := os.Stat(path)
  if err != nil {
    return false
  }
  return s.IsDir()
}

// create a temp file
func fileWrite(name string, buf []byte) (string, error) {
  tempDir := filepath.Join(os.TempDir(), "remote-clipboard")
  if !isDirExist(tempDir) {
    err := os.Mkdir(tempDir, 0666)
    if err != nil {
      return "", fmt.Errorf("failed to create temp dir: %w", err)
    }
  }

  tempFile := filepath.Join(tempDir, name)
  err := os.WriteFile(tempFile, buf, 0666)
  if err != nil {
    return "", fmt.Errorf("failed to write temp file: %w", err)
  }

  return tempFile, nil
}

func readFilePath() ([]byte, error) {
  hMem, _, err := getClipboardData.Call(cfHDROP)
  if hMem == 0 {
    return nil, err
  }
  p, _, err := globalLock.Call(hMem)
  if p == 0 {
    return nil, err
  }
  defer globalUnlock.Call(hMem)

  size := unsafe.Sizeof(uint16(0))

  //0x0014 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000 0x0000
  //0x0001 0x0000 filepath1 0x0000 filepath2 0x0000 0x0000
  basePos := uintptr(9)

  var filePath []string
  for ptr := unsafe.Pointer(p + basePos*size); ; {

    if *(*uint16)(ptr) == 0 && *(*uint16)(unsafe.Pointer(uintptr(ptr) + size)) == 0 {
      // End by NUL NUL
      break
    }

    // Split by NUL terminator
    if *(*uint16)(ptr) == 0 {
      len, temp := convertBufToStr(uintptr(ptr) + size)
      filePath = append(filePath, temp)
      ptr = unsafe.Pointer(uintptr(ptr) + uintptr(len)*size)
    } else {
      ptr = unsafe.Pointer(uintptr(ptr) + size)
    }
  }

  // Only read first file
  return fileRead(filePath[0])
}

func writeFilePath(name string, buf []byte) error {
  r, _, err := emptyClipboard.Call()
  if r == 0 {
    log.Errorln("Failed to clear clipboard:", err)
    return err
  }

  // empty text, we are done here.
  if len(buf) == 0 {
    return nil
  }

  // save temp file in temp dir
  tmpFile, err := fileWrite(name, buf)
  if err != nil {
    return err
  }

  s, err := syscall.UTF16FromString(tmpFile)
  if err != nil {
    log.Errorln("Failed to convert given string:", err)
    return err
  }

  additionLen := 11 * int(unsafe.Sizeof(s[0]))

  textLen := len(s) * int(unsafe.Sizeof(s[0]))

  hMem, _, err := globalAlloc.Call(gMemGHND, uintptr(textLen+additionLen))
  if hMem == 0 {
    log.Errorln("Failed to alloc global memory:", err)
    return err
  }

  pMem, _, err := globalLock.Call(hMem)
  if pMem == 0 {
    log.Errorln("Failed to lock global memory:", err)
    return err
  }
  defer globalUnlock.Call(hMem)

  *(*uint16)(unsafe.Pointer(pMem)) = 0x0014
  *(*uint16)(unsafe.Pointer(pMem + 8*unsafe.Sizeof(s[0]))) = 0x0001

  // no return value
  moveMemory.Call(pMem+uintptr(additionLen-2), uintptr(unsafe.Pointer(&s[0])), uintptr(textLen))

  handle, _, err := setClipboardData.Call(cfHDROP, hMem)
  if handle == 0 {
    globalFree.Call(hMem)
    log.Errorln("Failed to set text to clipboard:", err)
    return err
  }

  return nil
}

func watch(ctx context.Context) <-chan []byte {
  recv := make(chan []byte, 1)
  ready := make(chan struct{})
  go func() {
    ti := time.NewTicker(time.Second)
    clipboardCount, _, _ = getClipboardSequenceNumber.Call()
    ready <- struct{}{}
    for {
      select {
      case <-ctx.Done():
        close(recv)
        return
      case <-ti.C:
        countLock.Lock()
        cur, _, _ := getClipboardSequenceNumber.Call()
        if clipboardCount != cur {
          log.Debugf("Clipboard data changed, cur %d, last %d.", cur, clipboardCount)
          b, err := read()
          if b == nil || err != nil {
            log.Errorln("Failed to read:", err)
            clipboardCount = cur
            countLock.Unlock()
            continue
          }
          recv <- b
          clipboardCount = cur
        }
        countLock.Unlock()
      }
    }
  }()
  <-ready
  return recv
}

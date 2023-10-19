package clipboard

import (
  "context"
  "sync"

  log "github.com/sirupsen/logrus"
)

var (
  // guarantee one read at a time.
  lock = sync.Mutex{}
)

// Read returns a chunk of bytes of the clipboard data if it presents
// in the desired format t presents. Otherwise, it returns nil.
func Read() []byte {
  lock.Lock()
  defer lock.Unlock()

  buf, err := read()
  if err != nil {
    log.Errorf("Read clipboard err: %v.\n", err)
    return nil
  }
  return buf
}

// Write writes a given buffer to the clipboard in a specified format.
// Write returned a receive-only channel can receive an empty struct
// as a signal, which indicates the clipboard has been overwritten from
// this write.
func Write(buf []byte) <-chan struct{} {
  lock.Lock()
  defer lock.Unlock()

  changed, err := write(buf)
  if err != nil {
    log.Errorf("Write to clipboard err: %v\n", err)
    return nil
  }
  return changed
}

// Watch returns a receive-only channel that received the clipboard data
// whenever any change of clipboard data in the desired format happens.
//
// The returned channel will be closed if the given context is canceled.
func Watch(ctx context.Context) <-chan []byte {
  return watch(ctx)
}

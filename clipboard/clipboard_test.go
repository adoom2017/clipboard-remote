package clipboard

import (
    "os"
    "testing"
)

/* func TestRead(t *testing.T) {
    buf := Read()
    if buf == nil {
        t.Fatal("Read error.")
        return
    }

    clipInfo, _ := cm.DecodeToStruct(buf)

    switch clipInfo.Type {
    case cfDIBV5:
        t.Log("Clipboard data format is image.")
        return
    case cfHDROP:
        t.Log("Clipboard data format is file path.")
        os.WriteFile("D:\\clipboard\\"+clipInfo.Name, clipInfo.Buff, 0666)
        os.WriteFile("D:\\clipboard\\clip.bin", buf, 0666)
        return
    case cfUNICODETEXT:
        t.Log("content:", string(clipInfo.Buff))
        t.Log("Clipboard data format is text.")
        return
    default:
        t.Fatal("Clipboard data format is: ", clipInfo.Type)
        return
    }
} */

func TestWrite(t *testing.T) {
    buf, err := os.ReadFile("D:\\clipboard\\clip.bin")
    if err != nil {
        t.Fatal("Failed to read bin file:", err)
        return
    }

    Write(buf)
    //<-changed
    t.Log("Succeed to write clipboard.")
}

/* func TestWatch(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
    defer cancel()

    var lastRead []byte

    dataCh := Watch(ctx)

    for {
        select {
        case <-ctx.Done():
            if cm.BytesToString(lastRead) == "" {
                t.Fatalf("clipboard watch never receives a notification")
            }
            return
        case data, ok := <-dataCh:
            if !ok {
                if cm.BytesToString(lastRead) == "" {
                    t.Fatalf("clipboard watch never receives a notification")
                }
                return
            }
            clipInfo, _ := cm.DecodeToStruct(data)

            t.Log("Received data:", cm.BytesToString(clipInfo.Buff))
            lastRead = data
        }
    }
} */

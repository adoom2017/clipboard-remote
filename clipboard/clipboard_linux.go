package clipboard

import "context"

func read() (buf []byte, err error) {
  panic("clipboard: cannot use when CGO_ENABLED=0")
}

func write(buf []byte) (<-chan struct{}, error) {
  panic("clipboard: cannot use when CGO_ENABLED=0")
}

func watch(ctx context.Context) <-chan []byte {
  panic("clipboard: cannot use when CGO_ENABLED=0")
}

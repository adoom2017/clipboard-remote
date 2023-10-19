package utils

import (
  "bytes"
  "encoding/gob"
  "os"
)

type ClipType int

const (
  CLIP_TEXT ClipType = 0
  CLIP_PATH ClipType = 1
)

type ClipBoardBuff struct {
  Type ClipType
  Name string
  Buff []byte
}

type RespInfo struct {
  Code    int       `json:"code"`
  Message string    `json:"message"`
  Data    *DataInfo `json:"data,omitempty"`
}

type DataInfo struct {
  ClientID string `json:"client_id,omitempty"`
  Type     string `json:"type,omitempty"`
  Content  string `json:"content,omitempty"`
}

func EncodeToBytes(cb ClipBoardBuff) ([]byte, error) {

  buf := bytes.Buffer{}
  enc := gob.NewEncoder(&buf)
  err := enc.Encode(cb)
  if err != nil {
    return nil, err
  }

  return buf.Bytes(), nil
}

func DecodeToStruct(buf []byte) (ClipBoardBuff, error) {

  p := ClipBoardBuff{}
  dec := gob.NewDecoder(bytes.NewReader(buf))
  err := dec.Decode(&p)
  if err != nil {
    return p, err
  }
  return p, nil
}

func Exists(path string) bool {
  _, err := os.Stat(path)
  return !os.IsNotExist(err)
}

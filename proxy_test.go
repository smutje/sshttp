package sshttp

import (
  "io"
  "net"
  "testing"
)

func TestSimpleProxy(t *testing.T){
  b := Listen()
  go func(b net.Listener){
    c,err := b.Accept()
    defer c.Close()
    if err != nil {
      t.Fatal(err)
    }
    buf := make([]byte, 4)
    _,err = io.ReadFull(c, buf)
    if err != nil {
      t.Fatal(err)
    }
  }(b)
  l := Listen()
  srv := genServer(ProxyHandler(b),t)
  go srv.ServeNoClose(l)
  con, err := l.Connect()
  if err != nil {
    t.Fatal(err)
  }
  cl,err := Dial(con,&cConf)
  if err != nil {
    t.Fatal(err)
  }
  ch, _, err := cl.OpenChannel("ping",nil)
  io.WriteString(ch,"ping")
  ch.Close()
  cl.Close()
  l.Close()
  srv.Shutdown()
}


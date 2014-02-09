package sshttp

import (
  "net"
  "io"
  "code.google.com/p/gosshnew/ssh"
)

type Connector interface{
  Connect() (net.Conn, error)
}

func ProxyHandler(backend Connector) ChannelHandler {
  return func(srv ServerChannel, nch ssh.NewChannel){
    proxyChannel(backend, srv, nch)
  }
}

func proxyChannel(backend Connector, srv ServerChannel, nch ssh.NewChannel){
  src, err := backend.Connect()
  if err != nil {
    nch.Reject(ssh.ConnectionFailed, "backend connect failed")
    return
  }
  defer src.Close()
  dst, reqch, err := nch.Accept()
  if err != nil {
    srv.HandleError(err)
    return
  }
  go ssh.DiscardRequests(reqch)
  defer dst.Close()
  io.Copy(dst, src)
}


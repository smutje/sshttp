package sshttp

import (
  "net"
  "bytes"
  "code.google.com/p/gosshnew/ssh"
)

func Dial(conn net.Conn,config *ssh.ClientConfig) (*ssh.Client,error) {
  c, chans, reqs, err := ssh.NewClientConn(conn, conn.RemoteAddr().String(), config)
  if err != nil {
    return nil, err
  }
  return ssh.NewClient(c, chans, reqs), nil
}

type PublicKeyCallback func(conn ssh.ConnMetadata, algo string, pubkey []byte) bool

func AcceptPublicKey(keys ...ssh.PublicKey) PublicKeyCallback {
  return func(_ ssh.ConnMetadata, algo string, pubkey []byte) bool {
    for _, key := range(keys) {
      if key.PublicKeyAlgo() == algo && bytes.Compare(pubkey, ssh.MarshalPublicKey(key)) == 0 {
        return true
      }
    }
    return false
  }
}

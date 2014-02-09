package sshttp

import (
  "net"
  "bytes"
  "io"
  "os"
  "sync"
  "errors"
  "code.google.com/p/gosshnew/ssh"
)

var (
  DENIED = errors.New("Key denied")
)

func Dial(conn net.Conn,config *ssh.ClientConfig) (*ssh.Client,error) {
  c, chans, reqs, err := ssh.NewClientConn(conn, conn.RemoteAddr().String(), config)
  if err != nil {
    return nil, err
  }
  return ssh.NewClient(c, chans, reqs), nil
}

type PublicKeyCallback func(conn ssh.ConnMetadata, pubkey ssh.PublicKey) (*ssh.Permissions,error)
type HostKeyCallback func(hostname string, remote net.Addr, key ssh.PublicKey) error

func AcceptPublicKey(keys ...ssh.PublicKey) PublicKeyCallback {
  return func(_ ssh.ConnMetadata, pubkey ssh.PublicKey) (*ssh.Permissions,error) {
    for _, key := range(keys) {
      if bytes.Compare(pubkey.Marshal(), key.Marshal()) == 0 {
        return &ssh.Permissions{},nil
      }
    }
    return nil,DENIED
  }
}

func AcceptHostKey(keys ...ssh.PublicKey) HostKeyCallback {
  return func(_ string, _ net.Addr, pubkey ssh.PublicKey ) error{
    for _, key := range(keys) {
      if bytes.Compare(pubkey.Marshal(), key.Marshal()) == 0 {
        return nil
      }
    }
    return DENIED
  }
}

type AuthorizedKeyFile struct {
  Path string
  Keys []ssh.PublicKey

  mux sync.Mutex
}

func (a *AuthorizedKeyFile) Accept(conn ssh.ConnMetadata, pubkey ssh.PublicKey) (*ssh.Permissions,error) {
  return AcceptPublicKey(a.Keys...)(conn, pubkey)
}

func (a *AuthorizedKeyFile) Load() error {
  a.mux.Lock()
  defer a.mux.Unlock()
  file, err := os.Open(a.Path)
  if err != nil {
    return err
  }
  defer file.Close()
  r := &io.LimitedReader{ file, 10 * 1024 * 1024 }
  var buf bytes.Buffer
  _, err = io.Copy(&buf, r)
  if err != nil {
    return err
  }
  rest := buf.Bytes()
  var keys []ssh.PublicKey
  for len(rest) > 0 {
    var key ssh.PublicKey
    key,_,_,rest,err = ssh.ParseAuthorizedKey(rest)
    if err != nil {
      if err.Error() == "ssh: no key found" {
        break
      }
      return err
    }
    keys = append(keys,key)
  }
  a.Keys = keys
  return nil
}

func SignerFromFile(path string) (s ssh.Signer, err error){
  file, err := os.Open(path)
  if err != nil {
    return
  }
  defer file.Close()
  r := &io.LimitedReader{ file, 1024 * 1024 }
  var buf bytes.Buffer
  _, err = io.Copy(&buf, r)
  if err != nil {
    return
  }
  return ssh.ParsePrivateKey(buf.Bytes())
}

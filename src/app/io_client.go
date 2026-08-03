package app

import (
	"github.com/pkg/sftp"
	"io/ioutil"
	"os"
)

type IOClientType int

type FileLike interface {
	Name() string
	Stat() (os.FileInfo, error)
	Read([]byte) (int, error)
	Close() error
	Write(p []byte) (n int, err error)
}

type IOClient interface {
	Stat(file string) (os.FileInfo, error)
	Lstat(file string) (os.FileInfo, error)
	Mkdir(path string) error
	MkdirAll(path string) error
	Create(file string) (FileLike, error)
	Open(file string) (FileLike, error)
	ReadDir(file string) ([]os.FileInfo, error)
	Rename(oldname, newname string) error
	Remove(path string) error
}

// Local
type LocalIOClient struct {
}

func (client *LocalIOClient) Stat(file string) (os.FileInfo, error) {
	return os.Stat(file)
}

func (client *LocalIOClient) Lstat(file string) (os.FileInfo, error) {
	return os.Lstat(file)
}

func (client *LocalIOClient) Mkdir(path string) error {
	return os.Mkdir(path, 0755)
}

func (client *LocalIOClient) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func (client *LocalIOClient) Create(file string) (FileLike, error) {
	return os.Create(file)
}

func (client *LocalIOClient) Open(file string) (FileLike, error) {
	return os.Open(file)
}

func (client *LocalIOClient) ReadDir(file string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(file)
}

func (client *LocalIOClient) Rename(oldname, newname string) error {
	return os.Rename(oldname, newname)
}

func (client *LocalIOClient) Remove(path string) error {
	return os.Remove(path)
}

// SFTP(Remote)
type SftpIOClient struct {
	SftpClient *sftp.Client
}

func (client *SftpIOClient) Stat(file string) (os.FileInfo, error) {
	return client.SftpClient.Stat(file)
}

func (client *SftpIOClient) Lstat(file string) (os.FileInfo, error) {
	return client.SftpClient.Lstat(file)
}

func (client *SftpIOClient) Mkdir(path string) error {
	return client.SftpClient.Mkdir(path)
}

func (client *SftpIOClient) MkdirAll(path string) error {
	return client.SftpClient.MkdirAll(path)
}

func (client *SftpIOClient) Create(file string) (FileLike, error) {
	return client.SftpClient.Create(file)
}

func (client *SftpIOClient) Open(file string) (FileLike, error) {
	return client.SftpClient.Open(file)
}

func (client *SftpIOClient) ReadDir(file string) ([]os.FileInfo, error) {
	return client.SftpClient.ReadDir(file)
}

func (client *SftpIOClient) Rename(oldname, newname string) error {
	return client.SftpClient.Rename(oldname, newname)
}

func (client *SftpIOClient) Remove(path string) error {
	return client.SftpClient.Remove(path)
}

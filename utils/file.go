package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// CheckPathExists 检查文件目录是否存在
// 存在返回true,否则返回false
func CheckPathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}

	return !os.IsNotExist(err)
}

// CreatePath 递归创建文件
// error==nil,创建成功,否则创建失败
// 内部调用 MkdirAll
func CreatePath(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// CheckAndCreate 检查文件或者目录是否存在，不存在则创建
// error==nil,创建成功,否则创建失败
func CheckAndCreate(path string) error {
	if !CheckPathExists(path) {
		return CreatePath(path)
	}

	return nil
}

//GetDirFromPath 从path获取当前文件所在目录
// 返回目录路径
func GetDirFromPath(path string) string {
	return filepath.Dir(path)
}

//GetDirFromPath 从path获取当前文件名字
// 返回文件名字
func GetFileNameFromPath(path string) string {
	_, f := filepath.Split(path)
	return f
}

func WalkFiles(fileDir string, suffix ...string) ([]string, error) {
	rd, err := ioutil.ReadDir(fileDir)
	if err != nil {
		return nil, err
	}

	var out []string

	for _, fi := range rd {
		if !fi.IsDir() {
			if len(suffix) > 0 {
				for _, v := range suffix {
					if strings.HasSuffix(fi.Name(), v) {
						out = append(out, filepath.Join(fileDir, fi.Name()))
						break
					}
				}
			} else {
				out = append(out, filepath.Join(fileDir, fi.Name()))
			}
		}
	}

	return out, nil
}

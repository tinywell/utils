package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defDir           = "logs"
	defName          = "default.log"
	defCheckInterval = time.Second * 60 * 5
	dateFormat       = "2006-01-02"
)

// FileSplitType 日志文件的切分方式
type FileSplitType int

const (
	// STypeDate 按照日期切分
	STypeDate FileSplitType = iota
	// STypeSize 按照大小切分 单位 B
	STypeSize
	// STypeTime 按照时间切分
	STypeTime
)

// FileWriter 日志文件
type FileWriter struct {
	fileMutex     sync.Mutex
	file          *os.File
	fileName      string
	createTime    time.Time
	dir           string
	name          string
	splitType     FileSplitType
	checkInterval time.Duration
	splitSize     int64
	splitTime     time.Duration
}

// NewDateSplitWriter 返回 根据日期分割的 日志文件 writer
func NewDateSplitWriter() (*FileWriter, error) {
	w := &FileWriter{
		dir:           defDir,
		name:          defName,
		splitType:     STypeDate,
		checkInterval: defCheckInterval,
	}
	// w.newFile()
	return w, nil
}

// NewSizeSplitWriter 返回 根据文件大小(单位 Byte)分割的 日志文件 writer
func NewSizeSplitWriter(size int64) (*FileWriter, error) {
	w := &FileWriter{
		dir:           defDir,
		name:          defName,
		splitType:     STypeSize,
		checkInterval: defCheckInterval,
		splitSize:     size,
	}
	// w.newFile()
	return w, nil
}

// NewTimeSplitWriter 返回根据时间分割的 日志文件 writer
func NewTimeSplitWriter(d time.Duration) (*FileWriter, error) {
	w := &FileWriter{
		dir:           defDir,
		name:          defName,
		splitType:     STypeTime,
		checkInterval: defCheckInterval,
		splitTime:     d,
	}
	// w.newFile()
	return w, nil
}

// SyncWriter 刷新 wirter 文件相关配置
func (fw *FileWriter) SyncWriter() {
	fw.newFile()
}

// Write 实现 io.Writer 接口
func (fw *FileWriter) Write(p []byte) (n int, err error) {
	if fw.file == nil {
		fw.newFile()
	}
	return fw.file.Write(p)
}

// SetDir 设置日志目录，默认 ”logs“
func (fw *FileWriter) SetDir(dir string) {
	fw.dir = dir
}

// SetName 设置日志名字，默认 ”default.log“
func (fw *FileWriter) SetName(name string) {
	fw.name = name
}

// SetCheckInterval 设置日志分割检查间隔时间
func (fw *FileWriter) SetCheckInterval(d time.Duration) {
	fw.checkInterval = d
}

// StartCheck 启动分割检查
func (fw *FileWriter) StartCheck() {
	var ticker *time.Ticker
	if fw.splitType == STypeTime {
		ticker = time.NewTicker(fw.splitTime)
	} else {
		ticker = time.NewTicker(fw.checkInterval)
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				if fw.splitType == STypeTime {
					fw.split()
				} else if fw.checkSplit() {
					// fmt.Println("split")
					fw.split()
				}
			}
		}
	}()
}

func (fw *FileWriter) checkSplit() bool {
	switch fw.splitType {
	case STypeDate:
		cDate, _ := time.Parse(dateFormat, fw.createTime.Format(dateFormat))
		nDate, _ := time.Parse(dateFormat, time.Now().Format(dateFormat))
		// if nDate.Sub(cDate) == time.
		return nDate.After(cDate)

	case STypeSize:
		fileInfo, err := os.Stat(fw.fileName)
		if err != nil {
			return false
		}
		return fileInfo.Size() > fw.splitSize

	case STypeTime:
		return time.Now().Sub(fw.createTime) > fw.splitTime
	default:
		return false
	}
}

func (fw *FileWriter) split() {
	fw.fileMutex.Lock()
	defer fw.fileMutex.Unlock()
	fw.backup()
	fw.newFile()
}

func (fw *FileWriter) backup() {
	fw.file.Close()
	os.Rename(fw.fileName, fw.getBackupName())
}

func (fw *FileWriter) getBackupName() string {
	var count int
	name := fw.createTime.Format(dateFormat)
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(info.Name(), name) {
			count++
		}
		return nil
	}
	err := filepath.Walk(fw.dir, walkFunc)
	if err != nil {
		return filepath.Join(fw.dir, fmt.Sprintf("%s.x.log", name))
	}
	// fmt.Println("count:", count)
	return filepath.Join(fw.dir, fmt.Sprintf("%s.%03d.log", name, count))
}

func (fw *FileWriter) newFile() {
	if _, err := os.Stat(fw.dir); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(fw.dir, 0777)
			if err != nil {
				panic(err)
			}
		}
	}
	fw.fileName = filepath.Join(fw.dir, fw.name)
	file, err := os.Create(fw.fileName)
	if err != nil {
		panic(err)
	}
	fw.file = file
	fw.createTime = time.Now()
}

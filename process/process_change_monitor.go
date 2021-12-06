package process

import (
	"fmt"

	"github.com/ochinchina/filechangemonitor"
)

var fileChangeMonitor = filechangemonitor.NewFileChangeMonitor(10)

// addProgramChangeMonitor add a program change listener to monitor if the program binary
func addProgramChangeMonitor(path string, fileChangeCb func(path string, mode filechangemonitor.FileChangeMode)) {
	fileChangeMonitor.AddMonitorFile(path,
		false,
		filechangemonitor.NewExactFileMatcher(path),
		filechangemonitor.NewFileChangeCallbackWrapper(fileChangeCb),
		filechangemonitor.NewFileMD5CompareInfo())
}

// addConfigChangeMonitor add a program change listener to monitor if any one of its configuration files is changed
func addConfigChangeMonitor(path, filePattern string, fileChangeCb func(path string, mode filechangemonitor.FileChangeMode)) {
	fmt.Printf("filePattern=%s\n", filePattern)
	fileChangeMonitor.AddMonitorFile(path,
		true,
		filechangemonitor.NewPatternFileMatcher(filePattern),
		filechangemonitor.NewFileChangeCallbackWrapper(fileChangeCb),
		filechangemonitor.NewFileMD5CompareInfo())
}

package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"sync"
	"time"
	"strconv"
	"encoding/json"
	"github.com/mattn/go-colorable"
//	"github.com/mattn/go-isatty"
	"github.com/valyala/fasttemplate"

	"github.com/labstack/gommon/color"
)

type (
	Logger struct {
		prefix     string
		level      LEVEL
		output     io.Writer
		template   *fasttemplate.Template
		levels     []string
		color      *color.Color
		bufferPool sync.Pool
		mutex      *sync.Mutex
		file       *LogFileConfig
	}

	LEVEL uint8
	UNIT int64

	LogFileConfig struct {
		FileDir  string
		FileName string
		MaxCount int
		MaxSize  int64
		logfile  *os.File
		mu    *sync.RWMutex
		_suffix int
	}

	JSON map[string]interface{}
)

const (
	defaultFileName = "longo.log"
	TIME_RFC_CUSTOM= "2006-01-02T15:04:05.99999"
)

const (
	_ = iota
	KB UNIT = 1 << (iota * 10)
	MB
	GB
	TB
)

const (
	DEBUG LEVEL = iota
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

var (
	consoleOutput = true //同时控制台输出
	consoleTerminal = colorable.NewColorableStdout()

	global = New("-")

	//defaultFormat = "time=${time_rfc3339}, level=${level}, prefix=${prefix}, file=${short_file}, " +
	//"line=${line}, message=${message}\n"
        defaultFormat = "${prefix}|${time_custom}|${level}|${short_file}:${line} ${message}\n"
)

func New(prefix string) (l *Logger) {
	l = &Logger{
		level:    DEBUG,
		prefix:   prefix,
		template: l.newTemplate(defaultFormat),
		color:    color.New(),
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 256))
			},
		},
		mutex: new(sync.Mutex),
	}
	l.initLevels()
	l.SetOutput(consoleTerminal)
	return
}

func NewLoggerFile(prefix string, logfile LogFileConfig) (*Logger) {
	lg := New(prefix)
	lg.file = &logfile
	lg.file.mu= new(sync.RWMutex)

	logfile.mu.Lock()
	defer logfile.mu.Unlock()
	for i := 1; i <= int(logfile.MaxCount); i++ {
		if isExist(logfile.FileDir + "/" + logfile.FileName + "." + strconv.Itoa(i)) {
			logfile._suffix = i
		} else {
			break
		}
	}
	if !lg.file.isMustRename() {
		var err error
		lg.file.logfile, err = os.OpenFile(path.Join(logfile.FileDir, logfile.FileName), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			Fatal(err)
		}
		//lg.SetOutput(file)
	} else {
		lg.file.rename()
	}
	go logfile.fileMonitor()

	return lg
}

func (l *Logger) initLevels() {
	l.levels = []string{
		l.color.Blue("DEBUG"),
		l.color.Green("INFO"),
		l.color.Yellow("WARN"),
		l.color.Red("ERROR"),
		l.color.RedBg("FATAL"),
	}
}

func (l *Logger) newTemplate(format string) *fasttemplate.Template {
	return fasttemplate.New(format, "${", "}")
}

func (l *Logger) DisableColor() {
	l.color.Disable()
	l.initLevels()
}

func (l *Logger) EnableColor() {
	l.color.Enable()
	l.initLevels()
}

func (l *Logger) Prefix() string {
	return l.prefix
}

func (l *Logger) SetPrefix(p string) {
	l.prefix = p
}

func (l *Logger) Level() LEVEL {
	return l.level
}

func (l *Logger) SetLevel(v LEVEL) {
	l.level = v
}

func (l *Logger) Output() io.Writer {
	return l.output
}

func (l *Logger) SetFormat(f string) {
	l.template = l.newTemplate(f)
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.output = w
	/**
	  justin.h
	 */
	//if w, ok := w.(*os.File); !ok || !isatty.IsTerminal(w.Fd()) {
	//	l.DisableColor()
	//}
}

func (l *Logger) Print(i ...interface{}) {
	fmt.Fprintln(l.output, i...)
}

func (l *Logger) Printf(format string, args ...interface{}) {
	f := fmt.Sprintf("%s\n", format)
	fmt.Fprintf(l.output, f, args...)
}


func (l *Logger) Printj(j JSON) {
	json.NewEncoder(l.output).Encode(j)
}

func (l *Logger) Debug(i ...interface{}) {
	l.log(DEBUG, "", i...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Debugj(j JSON) {
	l.log(DEBUG, "json", j)
}

func (l *Logger) Info(i ...interface{}) {
	l.log(INFO, "", i...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Infoj(j JSON) {
	l.log(INFO, "json", j)
}



func (l *Logger) Warn(i ...interface{}) {
	l.log(WARN, "", i...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}


func (l *Logger) Warnj(j JSON) {
	l.log(WARN, "json", j)
}

func (l *Logger) Error(i ...interface{}) {
	l.log(ERROR, "", i...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func (l *Logger) Errorj(j JSON) {
	l.log(ERROR, "json", j)
}

func (l *Logger) Fatal(i ...interface{}) {
	l.log(FATAL, "", i...)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	os.Exit(1)
}

func (l *Logger) Fatalj(j JSON) {
	l.log(FATAL, "json", j)
}

func ConsoleOutput(b bool) {
	consoleOutput = b
}

func DisableColor() {
	global.DisableColor()
}

func EnableColor() {
	global.EnableColor()
}

func Prefix() string {
	return global.Prefix()
}

func SetPrefix(p string) {
	global.SetPrefix(p)
}

func Level() LEVEL {
	return global.Level()
}

func SetLevel(v LEVEL) {
	global.SetLevel(v)
}

func Output() io.Writer {
	return global.Output()
}

func SetOutput(w io.Writer) {
	global.SetOutput(w)
}

func SetFormat(f string) {
	global.SetFormat(f)
}

func Print(i ...interface{}) {
	global.Print(i...)
}

func Printf(format string, args ...interface{}) {
	global.Printf(format, args...)
}

func Printj(j JSON) {
	global.Printj(j)
}

func Debug(i ...interface{}) {
	global.Debug(i...)
}

func Debugf(format string, args ...interface{}) {
	global.Debugf(format, args...)
}

func Debugj(j JSON) {
	global.Debugj(j)
}

func Info(i ...interface{}) {
	global.Info(i...)
}

func Infof(format string, args ...interface{}) {
	global.Infof(format, args...)
}

func Infoj(j JSON) {
	global.Infoj(j)
}

func Warn(i ...interface{}) {
	global.Warn(i...)
}

func Warnf(format string, args ...interface{}) {
	global.Warnf(format, args...)
}

func Warnj(j JSON) {
	global.Warnj(j)
}

func Error(i ...interface{}) {
	global.Error(i...)
}

func Errorf(format string, args ...interface{}) {
	global.Errorf(format, args...)
}

func Errorj(j JSON) {
	global.Errorj(j)
}

func Fatal(i ...interface{}) {
	global.Fatal(i...)
}

func Fatalf(format string, args ...interface{}) {
	global.Fatalf(format, args...)
}

func Fatalj(j JSON) {
	global.Fatalj(j)
}

func (l *Logger) log(v LEVEL, format string, args ...interface{}) {
	//if w, ok := l.output.(*os.File); !ok || !isatty.IsTerminal(w.Fd()) {
	//	if consoleOutput {
	//		//当需要进行控制台输出,并且当前log不是输出到控制台时增加输出控制台
	//		l.EnableColor()
	//		l._log(consoleTerminal,v,format,args...)
	//	}
	//	l.DisableColor()
	//}else{
	//	l.EnableColor()
	//}
	//l._log(l.output,v,format,args...)
	if consoleOutput {
		l.EnableColor()
		l._log(consoleTerminal,v,format,args...)
	}
	if l.file!=nil{
		l.DisableColor()
		l._log(l.file.logfile,v,format,args...)
	}
}

func (l *Logger) _log(ioWriter io.Writer,v LEVEL, format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	buf := l.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer l.bufferPool.Put(buf)
	_, file, line, _ := runtime.Caller(3)

	if v >= l.level {
		message := ""
		if format == "" {
			message = fmt.Sprint(args...)
		} else {
			message = fmt.Sprintf(format, args...)
		}
		if v == FATAL {
			stack := make([]byte, 4 << 10)
			length := runtime.Stack(stack, true)
			message = message + "\n" + string(stack[:length])
		}
		_, err := l.template.ExecuteFunc(buf, func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "time_custom":
				return w.Write([]byte(time.Now().Format(TIME_RFC_CUSTOM)))
			case "time_rfc3339":
				return w.Write([]byte(time.Now().Format(time.RFC3339)))
			case "level":
				return w.Write([]byte(l.levels[v]))
			case "prefix":
				return w.Write([]byte(l.prefix))
			case "long_file":
				return w.Write([]byte(file))
			case "short_file":
				return w.Write([]byte(path.Base(file)))
			case "line":
				return w.Write([]byte(strconv.Itoa(line)))
			case "message":
				return w.Write([]byte(message))
			default:
				return w.Write([]byte(fmt.Sprintf("[unknown tag %s]", tag)))
			}
		})
		if err == nil {
			ioWriter.Write(buf.Bytes())
		}
	}
}
//l.output = w
//if w, ok := w.(*os.File); !ok || !isatty.IsTerminal(w.Fd()) {
//l.DisableColor()
//}
/////////////
func (f *LogFileConfig) isMustRename() bool {
	if f.MaxCount > 1 {
		if fileSize(path.Join(f.FileDir, f.FileName)) >= f.MaxSize {
			return true
		}
	}

	return false
}

func (f *LogFileConfig) rename() {
		f.coverNextOne()

}

func fileSize(file string) int64 {
	//	fmt.Println("fileSize", file)
	f, e := os.Stat(file)
	if e != nil {
		fmt.Println(e.Error())
		return 0
	}
	return f.Size()
}

func (f *LogFileConfig)fileMonitor() {
	timer := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-timer.C:
			f.fileCheck()
		}
	}
}

func (f *LogFileConfig)fileCheck() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	if f.isMustRename() {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.rename()
	}
}

func (f *LogFileConfig) nextSuffix() int {
	return int(f._suffix%int(f.MaxCount) + 1)
}

func (f *LogFileConfig) coverNextOne() {
	f._suffix = f.nextSuffix()
	if f.logfile != nil {
		f.logfile.Close()
	}
	if isExist(f.FileDir + "/" + f.FileName + "." + strconv.Itoa(int(f._suffix))) {
		os.Remove(f.FileDir + "/" + f.FileName + "." + strconv.Itoa(int(f._suffix)))
	}
	os.Rename(f.FileDir+"/"+f.FileName, f.FileDir+"/"+f.FileName+"."+strconv.Itoa(int(f._suffix)))
	f.logfile, _ = os.Create(f.FileDir + "/" + f.FileName)
}


func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
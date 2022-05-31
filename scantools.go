package scantools
import (
    "archive/tar"
    "archive/zip"
    "compress/gzip"
    "io"
    "os"
    "strings"
    "regexp"
    "path/filepath"
    "os/exec"
    "bufio"
    "io/ioutil"
    "log"
    "path"
    "github.com/hiaeia/scantools/utils"
)

const tempSecurtiyScan=".tempSecurtiyScan"

// 扫描任何东西
func HandleAnyThing(anyThing string) {
    fi, err := os.Stat(anyThing)

    if err == nil {
        if(fi.IsDir()) {
            // 目录,直接扫描
	    log.Printf("识别出目录 %s.", anyThing)
            ScanDir(anyThing)
	    return
        }
        // 压缩包
        var tarGzReg = regexp.MustCompile(`\.tar\.gz$`)
        if (tarGzReg.MatchString(anyThing)) {
	    log.Printf("识别出压缩包 %s.", anyThing)
            if err = DeCompressTarGz(anyThing, tempSecurtiyScan); err != nil {
                log.Printf(err.Error())
                return
            }
            ScanDir(tempSecurtiyScan)
            os.RemoveAll(tempSecurtiyScan)
            return
        }
        var zipReg = regexp.MustCompile(`\.zip$`)
        if (zipReg.MatchString(anyThing)) {
	    log.Printf("识别出压缩包 %s.", anyThing)
            if err = UnZip(anyThing, tempSecurtiyScan); err != nil {
                log.Printf(err.Error())
                return
            }
            ScanDir(tempSecurtiyScan)
            os.RemoveAll(tempSecurtiyScan)
            return
        }
        // 普通文件？
	log.Printf("识别出文件 %s.", anyThing)
        utils.ScanFile(anyThing)
        return

    } else {
        // git 地址
	var gitReg = regexp.MustCompile(`^https?://.*\.git$`)
	if (gitReg.MatchString(anyThing)) {
	    log.Printf("识别出git地址 %s.", anyThing)
	    _,err = execCommand("git", []string{"clone", anyThing, tempSecurtiyScan})
	    if err != nil {
                log.Printf(err.Error())
                return
	    }
            ScanDir(tempSecurtiyScan)
	    os.RemoveAll(tempSecurtiyScan)
            return
        }
        // wget 下载
	var wgetReg = regexp.MustCompile(`^https?://.*$`)
	if (wgetReg.MatchString(anyThing)) {
	    return
        }
	log.Println("Can't recognize which you provided")
        return
    }
}

var inGitFlag = false
var ignorePath = []string{`.git`, `.svn`}
var ignoreFile = []string{`.DS_Store`, `.gitignore`}

func ScanDir(dirname string) error {
    parentDirPath,_ := filepath.Abs(dirname)
    ExplorerRecursiveAndScanAndDelete(parentDirPath)
    return nil
}

// 递归遍历目录
func ExplorerRecursiveAndScanAndDelete(parentFilePath string) {
    // 取出节点的信息
    p, err := os.Stat(parentFilePath)
    if err != nil {
        log.Println(err.Error())
        return
    }

    // 非目录
    if !p.IsDir() {
        // 扫描单个文件
        utils.ScanFile(parentFilePath)
        return
    }

    // git 特殊处理
    if (!inGitFlag) {
        _, err = os.Stat(filepath.Join(parentFilePath ,".git"))
        if (err == nil) {// .git 存在
            inGitFlag = true
	    log.Printf("当前目录：%s", parentFilePath)
            // 在当前目录，切换不同分支，然后遍历
	    err = os.Chdir(parentFilePath)
	    if err != nil {
		log.Println(err.Error())
		return
	    }
	    lines, err := execCommand("bash", []string{"-c", `git branch -r | awk '{ print $1 }'`})
            if (err == nil) {
                for _,branch := range lines {//原封不同，再递归一次
                    _,err = execCommand("git", []string{"checkout", strings.Replace(branch, "\n", "", -1)})
                    if (err != nil) {
                        return
                    }
                    log.Printf("进入分支 %s...", branch)
                    // 相同的目录再处理一遍，这次不处理分支切换
                    ExplorerRecursiveAndScanAndDelete(parentFilePath)
                }
                inGitFlag = false
            }
        }
    }

    // log.Printf("正在扫描 %s 目录...", p.Name())
    // 目录中的文件和子目录
    items, err := ioutil.ReadDir(parentFilePath)
    if err != nil {
        log.Printf("%s:%s", "列举目录错误", err)
        return
    }

    for _, f := range items {
        childFilePath := path.Join(parentFilePath, f.Name())
	log.Printf("Found %s.", f.Name())

        if f.IsDir() { // 目录
            // 在忽略目录中的目录
            if IsInSlice(ignorePath, f.Name()) {
                continue
            }
            // 非忽略目录，进去递归
            ExplorerRecursiveAndScanAndDelete(childFilePath)
        } else { // 文件
            // 忽略文件
            if IsInSlice(ignoreFile, f.Name()) {
                continue
            }

            utils.ScanFile(childFilePath)
        }
    }
}

// IsInSlice 判断目标字符串是否是在切片中
func IsInSlice(slice []string, s string) bool {
    for _, f := range slice {
        if f == s {
            return true
        }
    }
    return false
}

// -----------------------------------------------------------------------------------
// 以下是辅助函数
func DeCompressTarGz(tarFile, destDir string) error {
    err := os.MkdirAll(destDir, os.ModePerm)
    if err != nil {
        return err
    }

    srcFile, err := os.Open(tarFile)
    if err != nil {
        return err
    }
    defer srcFile.Close()
    gr, err := gzip.NewReader(srcFile)
    if err != nil {
        return err
    }
    defer gr.Close()
    fr := tar.NewReader(gr)
    for {
        hdr, err := fr.Next()
        if err != nil {
            if err == io.EOF {
                break
            } else {
                return err
            }
        }
	//log.Printf("解压缩%s...", hdr.Name)
        filePath := filepath.Join(destDir, hdr.Name)
	if (hdr.Typeflag == tar.TypeDir){
	    err = os.MkdirAll(filePath, os.ModePerm)
	    if err != nil {
		return err
	    }
	    continue
	}
	// 其实只调用0次或者1次
	err = os.MkdirAll(string([]rune(filePath)[0:strings.LastIndex(filePath, "/")]), os.ModePerm)
	if err != nil {
	    return err
	}
	fw, err := os.Create(filePath)
        if err != nil {
            return err
        }

	//log.Printf("解压缩%s成功", filePath)
        _, err = io.Copy(fw, fr)
        if err != nil {
            fw.Close()
            return err
        }
    }
    return nil
}

func UnZip(src, destDir string) (err error) {
    err = os.MkdirAll(destDir, os.ModePerm)
    if err != nil {
        return err
    }

    zr, err := zip.OpenReader(src)
    defer zr.Close()
    if err != nil {
        return err
    }

    for _, file := range zr.File {
        path := filepath.Join(destDir, file.Name)

        if file.FileInfo().IsDir() {
            if err := os.MkdirAll(path, file.Mode()); err != nil {
                return err
            }
            continue
        }

        fr, err := file.Open()
        if err != nil {
            return err
        }

        fw, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, file.Mode())
        if err != nil {
            fr.Close()
            return err
        }

        _, err = io.Copy(fw, fr)
        if err != nil {
            fr.Close()
            fw.Close()
            return err
        }
    }

    return nil
}

// 返回多行输出，每行一个string
func execCommand(commandName string, params []string) ([]string, error) {
    lines := []string{}

    log.Println(commandName + " "  + strings.Join(params, " "))
    cmd := exec.Command(commandName, params...)

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        log.Println(err.Error())
        return  lines, err
    }
    // 输出错误流，git用
    cmd.Stderr = cmd.Stdout

    cmd.Start()
    reader := bufio.NewReader(stdout)

    for {
        line, err := reader.ReadString('\n')
	if err == io.EOF {
	    break
	}
        if err != nil {
            cmd.Wait()
	    return lines, err
        }
        log.Println(line)
        lines=append(lines, line)
    }

    cmd.Wait()
    return lines, nil
}

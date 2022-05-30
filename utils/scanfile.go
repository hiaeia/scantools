package utils

import (
	"strconv"
	"os"
	"io"
	"log"
	"fmt"
	"regexp"
	"bufio"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cdn"
	"github.com/dlclark/regexp2"
	"github.com/hiaeia/scantools/secret"
)

func ScanFile(filePath string) error{
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	//log.Printf("正在扫描:%s...\n", filePath)

	regexpSegment := regexp.MustCompile(`\b(?i)[a-zA-Z_\-\.]*secret\b`)
	regexpId := regexp.MustCompile(`\b(?i)[a-zA-Z_\-\.]*id\b`)

	var last = ""
	reader := bufio.NewReader(f)
	for {
		read, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}

		// log.Println(string(read))
		skMatched := regexpSegment.FindString(string(read))
		//log.Println(skMatched)
		//log.Println(len(string(read)))
		if (len(skMatched) != 0) {
			peek, _ := reader.Peek(256)
			content := last + string(read) + string(peek)
			last = string(read)

			//log.Println(last)
			//log.Println(content)
			akMatched := regexpId.FindString(content)
			if (len(akMatched) == 0) {
				continue
			}

			regexpAKBytes,_ := regexp2.Compile(`(?<=` + akMatched + `( *)(=|:)( *)["']{0,1})(:?[a-zA-Z0-9]{26}|[a-zA-Z0-9]{24}|(?:LT)?RSA\.[a-zA-Z0-9]{16}|[a-zA-Z0-9]{16})(?=["']{0,1})`, 0)
			regexpSKBytes,_ := regexp2.Compile(`(?<=` + skMatched + `( *)(=|:)( *)["']{0,1})[a-zA-Z0-9]{30}(?=["']{0,1})`, 0)

			//log.Println(regexpSK.FindAllString(content, -1))
			suspiciousAK,_ := regexpAKBytes.FindStringMatch(content)
			suspiciousSK,_ := regexpSKBytes.FindStringMatch(content)

			if (suspiciousAK == nil){
				continue
			}
			if (suspiciousSK == nil){
				continue
			}

			// log.Println(akMatched + ":" + suspiciousAK.String())
			// log.Println(skMatched + ":" + suspiciousSK.String())

			ret := ISAliyunAK(suspiciousAK.String(), suspiciousSK.String())
			if (ret == 1) {
				log.Printf("\033[1;33;40mFound:\033[0m\n\tAK:%s.\n", suspiciousAK.String())
				msg := fmt.Sprintf("Found:\n\tAK:%s.\n", suspiciousAK.String())
				secret.Submit2Dingding(msg)
			}
			if (ret == 2) {
				log.Printf("\033[1;33;40mFound:\033[0m\n\tAK:%s, SK:%s\n", suspiciousAK.String(), suspiciousSK.String())
				msg := fmt.Sprintf("Found:\n\tAK:%s, SK:%s\n", suspiciousAK.String(), suspiciousSK.String())
				secret.Submit2Dingding(msg)
			}
		}
		last = string(read)
	}

	log.Printf("扫描:%s 完成\n", filePath)
	return nil
}

func ISAliyunAK(ak string, sk string) int64 {
	client, err := cdn.NewClientWithAccessKey("cn-hangzhou", ak, sk)

	request := cdn.CreateDescribeCdnRegionAndIspRequest()
	request.Scheme = "https"

	_, err = client.DescribeCdnRegionAndIsp(request)
	if err != nil {
		matched, _ := regexp.MatchString("InvalidAccessKeyId.NotFound", err.Error())
		if matched {
			return 0
		}
		matched, _ = regexp.MatchString("SignatureDoesNotMatch", err.Error())
		if matched {
			return 1
		}
	}
	return 2
}

func test_ScanFile(filePath string) {
	ScanFile(filePath)
}

/*
func main() {
	var filePath string
	flag.StringVar(&filePath, "f", "", "文件名,默认为空")

	flag.Parse()
	ScanFile(filePath)
}
*/

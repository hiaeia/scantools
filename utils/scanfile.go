package utils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"

	 ecs20140526  "github.com/alibabacloud-go/ecs-20140526/v2/client"
	 openapi  "github.com/alibabacloud-go/darabonba-openapi/client"
	 util  "github.com/alibabacloud-go/tea-utils/service"
	"github.com/dlclark/regexp2"
	"github.com/hiaeia/scantools/secret"
)

func ScanFile(filePath string) []string {
	var result []string
	f, err := os.Open(filePath)
	if err != nil {
		log.Println(err.Error())
		return result
	}
	defer f.Close()

	//log.Printf("正在扫描:%s...\n", filePath)

	regexpSegment := regexp.MustCompile(`\b(?i)[a-zA-Z_\-\.]*secret\b`)
	regexpId := regexp.MustCompile(`\b(?i)[a-zA-Z_\-\.]*(id|key)\b`)

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
		if len(skMatched) != 0 {
			peek, _ := reader.Peek(256)
			content := last + string(read) + string(peek)
			last = string(read)

			//log.Println(last)
			//log.Println(content)
			akMatched := regexpId.FindString(content)
			if len(akMatched) == 0 {
				continue
			}

			regexpAKBytes, _ := regexp2.Compile(`(?<=`+akMatched+`( *)(=|:)( *)["']{0,1})(:?[a-zA-Z0-9]{26}|[a-zA-Z0-9]{24}|(?:LT)?RSA\.[a-zA-Z0-9]{16}|[a-zA-Z0-9]{16})(?=["']{0,1})`, 0)
			regexpSKBytes, _ := regexp2.Compile(`(?<=`+skMatched+`( *)(=|:)( *)["']{0,1})[a-zA-Z0-9]{30}(?=["']{0,1})`, 0)

			//log.Println(regexpSK.FindAllString(content, -1))
			suspiciousAK, _ := regexpAKBytes.FindStringMatch(content)
			suspiciousSK, _ := regexpSKBytes.FindStringMatch(content)

			if suspiciousAK == nil {
				continue
			}
			if suspiciousSK == nil {
				continue
			}

			log.Println(akMatched + ":" + suspiciousAK.String())
			log.Println(skMatched + ":" + suspiciousSK.String())

			ret := ISAliyunAK(suspiciousAK.String(), suspiciousSK.String())
			if ret == 1 {
				log.Printf("\033[1;33;40mFound:\033[0m\n\tAK:%s.\n", suspiciousAK.String())
				result = append(result, fmt.Sprintf("AK:%s, SK:XXXXXX", suspiciousAK.String()))
				msg := fmt.Sprintf("Found:\n\tAK:%s.\n", suspiciousAK.String())
				secret.Submit2Dingding(msg)
			}
			if ret == 2 {
				log.Printf("\033[1;33;40mFound:\033[0m\n\tAK:%s, SK:XXXXXX\n", suspiciousAK.String())
				result = append(result, fmt.Sprintf("AK:%s, SK:XXXXXX", suspiciousAK.String()))
				msg := fmt.Sprintf("Found:\nAK:%s, SK:XXXXXX", suspiciousAK.String())
				secret.Submit2Dingding(msg)
			}
		}
		last = string(read)
	}

	log.Printf("扫描:%s 完成\n", filePath)
	return result
}

func ISAliyunAK (accessKeyId string, accessKeySecret string) int64 {
	config := &openapi.Config{
		AccessKeyId: &accessKeyId,
		AccessKeySecret: &accessKeySecret,
	}

	endpoint := "ecs-cn-hangzhou.aliyuncs.com"
	config.Endpoint = &endpoint
	client := &ecs20140526.Client{}
	client, _ = ecs20140526.NewClient(config)

	describeRegionsRequest := &ecs20140526.DescribeRegionsRequest{}
	runtime := &util.RuntimeOptions{}
	_, err := client.DescribeRegionsWithOptions(describeRegionsRequest, runtime)
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

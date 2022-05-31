Demo代码
```
package main

import (
	"github.com/hiaeia/scantools"
	"flag"
)

func main() {
    var anything string
    flag.StringVar(&anything, "f", "", "文件名,默认为空")

    flag.Parse()
    
    scantools.HandleAnyThing(anything)
    // scantools.HandleAnyThing("/home/lighthouse/test_scan.tar.gz")
    // scantools.HandleAnyThing("/home/lighthouse/JxrApp.zip")
    // scantools.HandleAnyThing("/home/lighthouse/JxrApp.zip")
    // scantools.HandleAnyThing("/home/lighthouse/SecurityTest/mytools/test_scan")
}
```

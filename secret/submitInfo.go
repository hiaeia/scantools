package secret

import (
    "log"
    "github.com/hiaeia/send2ding"
)


func Submit2Dingding(msg string) {
    secret := "your secret"
    token  := "your token"
    content := `{"msgtype": "markdown","markdown": {"title":"AK/SK", "text": "`+ msg + `"}}`

    client := send2ding.New(token, secret)
    if err := client.Send(send2ding.TextMessage(content)); err != nil {
	log.Println(err.Error())
    }




}

package layer2

import (
	"fmt"
	"time"
)

//call define
func call(msg interface{}) {
	fmt.Printf("\ndccp call: msg = %v\n", msg)
}

func dccpcall(msg interface{}) <-chan string {
	ch := make(chan string, 800)
	fmt.Printf("\ndccp dccpcall: msg=%v\n", msg)
	Dccprotocol_broadcastToGroup(msg.(string))
	ch <- msg.(string)
	return ch
}

func dccpcallret(msg interface{}) {
	fmt.Printf("dccp dccpcallret: msg=%v\n", msg)
}

func Dccprotocol_startTest() {
	fmt.Printf("\n\nDCCP P2P test ...\n\n")
	Dccprotocol_registerCallback(call)
	Dccprotocol_registerMsgCallback(dccpcall)
	Dccprotocol_registerMsgRetCallback(dccpcallret)

	time.Sleep(time.Duration(10) * time.Second)

	select {} // note for client, or for server

	var num int = 0
	for {
		fmt.Printf("\nSendToDccpGroup ...\n")
		num += 1
		msg := fmt.Sprintf("%+v test SendToDccpGroup ...", num)
		Dccprotocol_sendToGroup(msg)
		time.Sleep(time.Duration(2) * time.Second)
	}
	select {}
}


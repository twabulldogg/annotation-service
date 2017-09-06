package search

import (
	"bytes"
	"errors"
	"log"
	"net"

	"github.com/m-lab/annotation-service/parser"
)

// TODO: Add a prometheus metric for when we can't find the IP
// Returns a parser.IPNode with the smallet range that includes the provided IP address
func SearchList(list []parser.IPNode, ipLookUp string) (parser.IPNode, error) {
	inRange := false
	var lastNodeIndex int
	userIP := net.ParseIP(ipLookUp)
	if userIP == nil {
		log.Println("Inputed IP string could not be parsed to net.IP")
		return parser.IPNode{}, errors.New("Invalid search IP")
	}
	for i := range list {
		if bytes.Compare(userIP, list[i].IPAddressLow) >= 0 && bytes.Compare(userIP, list[i].IPAddressHigh) <= 0 {
			inRange = true
			lastNodeIndex = i
		} else if inRange && bytes.Compare(userIP, list[i].IPAddressLow) < 0 {
			return list[lastNodeIndex], nil
		}
	}
	if inRange {
		return list[lastNodeIndex], nil
	}
	return parser.IPNode{}, errors.New("Node not found\n")
}

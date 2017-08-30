// Only files including IPv4, IPv6, and Location (in english)
// will be read and parsed into lists.
package parser

import (
	"encoding/binary"
	"errors"
	"log"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
)

const mapMax = 200000

// IPNode defines IPv4 and IPv6 databases
type IPNode struct {
	IPAddressLow  net.IP
	IPAddressHigh net.IP
	LocationIndex int // Index to slice of locations
	PostalCode    string
	Latitude      float64
	Longitude     float64
}

// LocationNode defines Location databases
type LocationNode struct {
	GeonameID     int
	ContinentCode string
	CountryCode   string
	CountryName   string
	MetroCode     int64
	CityName      string
}

// The GeoDataset struct bundles all the data needed to search and
// find data into one common structure
type GeoDataset struct {
	IP4Nodes      []IPNode       // The IPNode list containing IP4Nodes
	IP6Nodes      []IPNode       // The IPNode list containing IP6Nodes
	LocationNodes []LocationNode // The location nodes corresponding to the IPNodes
}


// Verify column length
func checkColumnLength(record []string, size int) error {
	if len(record) != size {
		log.Println("Incorrect number of columns in IP list", size, " got: ", len(record), record)
		return errors.New("Corrupted Data: wrong number of columns")
	}
	return nil
}

// Converts integer to net.IPv4
func Int2ip(str string) (net.IP, error) {
	num, err := strconv.Atoi(str)
	if err != nil {
		log.Println("Provided IP should be a number")
		return nil, errors.New("Inputed string cannot be converted to a number")
	}
	ft := float64(num)
	if ft > math.Pow(2, 32) || num < 1 {
		log.Println("Provided IP should be in the range of 0.0.0.1 and 255.255.255.255 ", str)
	}
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, uint32(num))
	ip = net.IPv4(ip[0], ip[1], ip[2], ip[3])
	return ip, nil
}

// Finds the smallest and largest net.IP from a CIDR range
// Example: "1.0.0.0/24" -> 1.0.0.0 , 1.0.0.255
func RangeCIDR(cidr string) (net.IP, net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, nil, errors.New("Invalid CIDR IP range")
	}
	lowIp := make(net.IP, len(ip))
	copy(lowIp, ip)
	mask := ipnet.Mask
	for x, _ := range ip {
		if len(mask) == 4 {
			if x < 12 {
				ip[x] |= 0
			} else {
				ip[x] |= ^mask[x-12]
			}
		} else {
			ip[x] |= ^mask[x]
		}
	}
	return lowIp, ip, nil
}

// Finds provided geonameID within idMap and returns the index in idMap
// locationIdMap := map[int]int{
//	609013: 0,
//	104084: 4,
//	17:     4,
// }
// lookupGeoId("17",locationIdMap) would return (2,nil).
// TODO: Add error metrics
func lookupGeoId(gnid string, idMap map[int]int) (int, error) {
	geonameId, err := strconv.Atoi(gnid)
	if err != nil {
		return 0, errors.New("Corrupted Data: geonameID should be a number")
	}
	loadIndex, ok := idMap[geonameId]
	if !ok {
		log.Println("geonameID not found ", geonameId)
		return 0, errors.New("Corrupted Data: geonameId not found")
	}
	return loadIndex, nil
}

func stringToFloat(str, field string) (float64, error) {
	flt, err := strconv.ParseFloat(str, 64)
	if err != nil {
		if len(str) > 0 {
			log.Println(field, " was not a number")
			output := strings.Join([]string{"Corrupted Data: ", field, " should be an int"}, "")
			return 0, errors.New(output)
		}
	}
	return flt, nil
}

func checkAllCaps(str, field string) (string, error) {
	match, _ := regexp.MatchString("^[0-9A-Z]*$", str)
	if match {
		return str, nil
	} else {
		log.Println(field, "should be all capitals and no numbers")
		output := strings.Join([]string{"Corrupted Data: ", field, " should be all caps"}, "")
		return "", errors.New(output)

	}
}

// Returns nil if two nodes are equal
// Used by the search package
func IsEqualIPNodes(expected, node IPNode) error {
	if !((node.IPAddressLow).Equal(expected.IPAddressLow)) {
		output := strings.Join([]string{"IPAddress Low inconsistent\ngot:", node.IPAddressLow.String(), " \nwanted:", expected.IPAddressLow.String()}, "")
		log.Println(output)
		return errors.New(output)
	}
	if !((node.IPAddressHigh).Equal(expected.IPAddressHigh)) {
		output := strings.Join([]string{"IPAddressHigh inconsistent\ngot:", node.IPAddressHigh.String(), " \nwanted:", expected.IPAddressHigh.String()}, "")
		log.Println(output)
		return errors.New(output)
	}
	if node.LocationIndex != expected.LocationIndex {
		output := strings.Join([]string{"LocationIndex inconsistent\ngot:", strconv.Itoa(node.LocationIndex), " \nwanted:", strconv.Itoa(expected.LocationIndex)}, "")
		log.Println(output)
		return errors.New(output)
	}
	if node.PostalCode != expected.PostalCode {
		output := strings.Join([]string{"PostalCode inconsistent\ngot:", node.PostalCode, " \nwanted:", expected.PostalCode}, "")
		log.Println(output)
		return errors.New(output)
	}
	if node.Latitude != expected.Latitude {
		output := strings.Join([]string{"Latitude inconsistent\ngot:", floatToString(node.Latitude), " \nwanted:", floatToString(expected.Latitude)}, "")
		log.Println(output)
		return errors.New(output)
	}
	if node.Longitude != expected.Longitude {
		output := strings.Join([]string{"Longitude inconsistent\ngot:", floatToString(node.Longitude), " \nwanted:", floatToString(expected.Longitude)}, "")
		log.Println(output)
		return errors.New(output)
	}
	return nil
}

func floatToString(num float64) string {
	return strconv.FormatFloat(num, 'f', 6, 64)
}

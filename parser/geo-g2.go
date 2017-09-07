package parser

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"

	"github.com/m-lab/annotation-service/loader"
)

const (
	ipNumColumnsGlite2        = 10
	locationNumColumnsGlite2  = 13
	gLite2Prefix              = "GeoLite2-City"
	geoLite2BlocksFilenameIP4 = "GeoLite2-City-Blocks-IPv4.csv"  // Filename of ipv4 blocks file
	geoLite2BlocksFilenameIP6 = "GeoLite2-City-Blocks-IPv6.csv"  // Filename of ipv6 blocks file
	geoLite2LocationsFilename = "GeoLite2-City-Locations-en.csv" // Filename of locations file
)

func LoadGeoLite2(zip *zip.Reader) (*GeoDataset, error) {
	locations, err := loader.FindFile(geoLite2LocationsFilename, zip)
	if err != nil {
		return nil, err
	}
	// geoidMap is just a temporary map that will be discarded once the blocks are parsed
	locationNode, geoidMap, err := LoadLocListGLite2(locations)
	if err != nil {
		return nil, err
	}
	blocks4, err := loader.FindFile(geoLite2BlocksFilenameIP4, zip)
	if err != nil {
		return nil, err
	}
	ipNodes4, err := LoadIPListGLite2(blocks4, geoidMap)
	if err != nil {
		return nil, err
	}
	blocks6, err := loader.FindFile(geoLite2BlocksFilenameIP6, zip)
	if err != nil {
		return nil, err
	}
	ipNodes6, err := LoadIPListGLite2(blocks6, geoidMap)
	if err != nil {
		return nil, err
	}
	return &GeoDataset{IP4Nodes: ipNodes4, IP6Nodes: ipNodes6, LocationNodes: locationNode}, nil
}

// Finds the smallest and largest net.IP from a CIDR range
// Example: "1.0.0.0/24" -> 1.0.0.0 , 1.0.0.255
func rangeCIDR(cidr string) (net.IP, net.IP, error) {
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

// Create Location list for GLite2 databases
func LoadLocListGLite2(reader io.Reader) ([]LocationNode, map[int]int, error) {
	idMap := make(map[int]int, mapMax)
	list := []LocationNode{}
	r := csv.NewReader(reader)
	// Skip the first line
	_, err := r.Read()
	if err == io.EOF {
		log.Println("Empty input data")
		return nil, nil, errors.New("Empty input data")
	}
	// FieldsPerRecord is the expected column length
	r.FieldsPerRecord = locationNumColumnsGlite2
	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			if len(record) != r.FieldsPerRecord {
				log.Println("Incorrect number of columns in IP list got: ", len(record), " wanted: ", r.FieldsPerRecord)
				return nil, nil, errors.New("Corrupted Data: wrong number of columns")

			} else {
				log.Println(err, ": ", record)
				return nil, nil, err
			}
		}
		var lNode LocationNode
		lNode.GeonameID, err = strconv.Atoi(record[0])
		if err != nil {
			if len(record[0]) > 0 {
				log.Println("GeonameID should be a number ", record[0])
				return nil, nil, errors.New("Corrupted Data: GeonameID should be a number")
			}
		}
		lNode.ContinentCode, err = checkCaps(record[2], "Continent code")
		if err != nil {
			return nil, nil, err
		}
		lNode.CountryCode, err = checkCaps(record[4], "Country code")
		if err != nil {
			return nil, nil, err
		}
		match, _ := regexp.MatchString(`^[^0-9]*$`, record[5])
		if match {
			lNode.CountryName = record[5]
		} else {
			log.Println("Country name should be letters only : ", record[5])
			return nil, nil, errors.New("Corrupted Data: country name should be letters")
		}
		lNode.MetroCode, err = strconv.ParseInt(record[11], 10, 64)
		if err != nil {
			if len(record[11]) > 0 {
				log.Println("MetroCode should be a number")
				return nil, nil, errors.New("Corrupted Data: metrocode should be a number")
			}
		}
		lNode.CityName = record[10]
		list = append(list, lNode)
		idMap[lNode.GeonameID] = len(list) - 1
	}
	return list, idMap, nil
}

// Creates a List of IPNodes
func LoadIPListGLite2(reader io.Reader, idMap map[int]int) ([]IPNode, error) {
	list := []IPNode{}
	r := csv.NewReader(reader)
	stack := []IPNode{}
	// Skip first line
	_, err := r.Read()
	if err == io.EOF {
		log.Println("Empty input data")
		return nil, errors.New("Empty input data")
	}
	var newNode IPNode
	for {
		// Example:
		// GLite2 : record = [2a04:97c0::/29,2658434,2658434,0,0,47,8,100]
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		err = checkNumColumns(record, ipNumColumnsGlite2)
		if err != nil {
			return nil, err
		}
		lowIp, highIp, err := rangeCIDR(record[0])
		if err != nil {
			return nil, err
		}
		newNode.IPAddressLow = lowIp
		newNode.IPAddressHigh = highIp
		// Look for GeoId within idMap and return index
		index, err := lookupGeoId(record[1], idMap)
		if err != nil {
			if backupIndex, err := lookupGeoId(record[2], idMap); err == nil {
				index = backupIndex
			} else {
				log.Println("Couldn't get a valid Geoname id!", record)
				//TODO: Add a prometheus metric here
			}

		}
		newNode.LocationIndex = index
		newNode.PostalCode = record[6]
		newNode.Latitude, err = stringToFloat(record[7], "Latitude")
		if err != nil {
			return nil, err
		}
		newNode.Longitude, err = stringToFloat(record[8], "Longitude")
		if err != nil {
			return nil, err
		}
		// Stack is not empty aka we're in a nested IP
		if len(stack) != 0 {
			// newNode is no longer inside stack's nested IP's
			if lessThan(stack[len(stack)-1].IPAddressHigh, newNode.IPAddressLow) {
				// while closing nested IP's
				for len(stack) > 0 {
					var pop IPNode
					pop, stack = stack[len(stack)-1], stack[:len(stack)-1]
					if len(stack) == 0 {
						break
					}
					peek := stack[len(stack)-1]
					if lessThan(newNode.IPAddressLow, peek.IPAddressHigh) {
						// if theres a gap inbetween imediately nested IP's
						if len(stack) > 0 {
							//log.Println("current stack: ",stack)
							//complete the gap
							peek.IPAddressLow = PlusOne(pop.IPAddressHigh)
							peek.IPAddressHigh = minusOne(newNode.IPAddressLow)
							list = append(list, peek)
						}
						break
					}
					peek.IPAddressLow = PlusOne(pop.IPAddressHigh)
					list = append(list, peek)
				}
			} else {
				// if we're nesting IP's
				// create begnning bounds
				lastListNode := &list[len(list)-1]
				lastListNode.IPAddressHigh = minusOne(newNode.IPAddressLow)

			}
		}
		stack = append(stack, newNode)
		list = append(list, newNode)
		newNode.IPAddressLow = newNode.IPAddressHigh
		newNode.IPAddressHigh = net.IPv4(255, 255, 255, 255)

	}
	return list, nil
}

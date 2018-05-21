package main

import "flag"
import "fmt"
import "strings"
import "net"
import "sort"
import "encoding/json"
import "strconv"
import "os"

type admin_info map[string]interface{}

func main() {
	server := flag.String("endpoint", "localhost:9001", "Admin socket endpoint")
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("usage:", os.Args[0], "[-endpoint=localhost:9001] command [key=value] [...]")
		fmt.Println("example:", os.Args[0], "getPeers")
		fmt.Println("example:", os.Args[0], "setTunTap name=auto mtu=1500 tap_mode=false")
		fmt.Println("example:", os.Args[0], "-endpoint=localhost:9001 getDHT")
		return
	}

	conn, err := net.Dial("tcp", *server)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	send := make(admin_info)
	recv := make(admin_info)

	for c, a := range args {
		if c == 0 {
			send["request"] = a
			continue
		}
		tokens := strings.Split(a, "=")
		if i, err := strconv.Atoi(tokens[1]); err == nil {
			send[tokens[0]] = i
		} else {
			switch tokens[1] {
			case "true":
				send[tokens[0]] = true
			case "false":
				send[tokens[0]] = false
			default:
				send[tokens[0]] = tokens[1]
			}
		}
	}

	if err := encoder.Encode(&send); err != nil {
		panic(err)
	}
	if err := decoder.Decode(&recv); err == nil {
		if recv["status"] == "error" {
			if err, ok := recv["error"]; ok {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Unspecified error occured")
			}
			os.Exit(1)
		}
		if _, ok := recv["request"]; !ok {
			fmt.Println("Missing request in response (malformed response?)")
			return
		}
		if _, ok := recv["response"]; !ok {
			fmt.Println("Missing response body (malformed response?)")
			return
		}
		req := recv["request"].(map[string]interface{})
		res := recv["response"].(map[string]interface{})

		switch req["request"] {
		case "dot":
			fmt.Println(res["dot"])
		case "help", "getPeers", "getSwitchPeers", "getDHT", "getSessions":
			maxWidths := make(map[string]int)
			var keyOrder []string
			keysOrdered := false

			for _, tlv := range res {
				for slk, slv := range tlv.(map[string]interface{}) {
					if !keysOrdered {
						for k := range slv.(map[string]interface{}) {
							keyOrder = append(keyOrder, fmt.Sprint(k))
						}
						sort.Strings(keyOrder)
						keysOrdered = true
					}
					for k, v := range slv.(map[string]interface{}) {
						if len(fmt.Sprint(slk)) > maxWidths["key"] {
							maxWidths["key"] = len(fmt.Sprint(slk))
						}
						if len(fmt.Sprint(v)) > maxWidths[k] {
							maxWidths[k] = len(fmt.Sprint(v))
							if maxWidths[k] < len(k) {
								maxWidths[k] = len(k)
							}
						}
					}
				}

				if len(keyOrder) > 0 {
					fmt.Printf("%-"+fmt.Sprint(maxWidths["key"])+"s  ", "")
					for _, v := range keyOrder {
						fmt.Printf("%-"+fmt.Sprint(maxWidths[v])+"s  ", v)
					}
					fmt.Println()
				}

				for slk, slv := range tlv.(map[string]interface{}) {
					fmt.Printf("%-"+fmt.Sprint(maxWidths["key"])+"s  ", slk)
					for _, k := range keyOrder {
						fmt.Printf("%-"+fmt.Sprint(maxWidths[k])+"s  ", fmt.Sprint(slv.(map[string]interface{})[k]))
					}
					fmt.Println()
				}
			}
		case "getTunTap", "setTunTap":
			for k, v := range res {
				fmt.Println("Interface name:", k)
				if mtu, ok := v.(map[string]interface{})["mtu"].(float64); ok {
					fmt.Println("Interface MTU:", mtu)
				}
				if tap_mode, ok := v.(map[string]interface{})["tap_mode"].(bool); ok {
					fmt.Println("TAP mode:", tap_mode)
				}
			}
		default:
			if json, err := json.MarshalIndent(recv["response"], "", "  "); err == nil {
				fmt.Println(string(json))
			}
		}
	}

	if v, ok := recv["status"]; ok && v == "error" {
		os.Exit(1)
	}
	os.Exit(0)
}

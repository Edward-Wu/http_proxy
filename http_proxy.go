package main

import (
	//"bytes"
	"fmt"
	"io"
	"log"
	"net"
	//"net/url"
	"strings"
    "os"
    "os/signal"
    "strconv"
    //"crypto/aes"
    //"crypto/cipher"
    //"time"
    //"errors"
    //"crypto/rand"
    //"encoding/base64"
    "encoding/binary"
)

var signChannel       = make(chan os.Signal, 1)

func usage() {
    fmt.Println("http_proxy :")
    fmt.Println("    -s serverip/domain name")
}

type paramsArgs struct {
    server_name   string   //srt server hostname
}


func parseArgs(pa *paramsArgs) (bool){
    i := 1
    for (i < len(os.Args)) {
        if os.Args[i] == "-s" {
            i ++
            pa.server_name = os.Args[i]
            i ++
        } else if os.Args[i] == "-h" {
            i ++
            usage()
            return false
        //} else if os.Args[i] == "-p" {
        } else {
            fmt.Printf("wrong input parameter '%s'.\n", os.Args[i])
            return false
        }
    }
    return true
}
var pa paramsArgs


func main() {

    log.SetFlags(log.LstdFlags|log.Lshortfile)
    var http_port = 80
    var https_port = 443
    var httpListener *net.TCPListener = nil
    var httpsListener *net.TCPListener = nil

    defer func() {
        fmt.Printf("main, closed listener.\n")	
        if httpListener != nil {
            httpListener.Close()
        }
        if httpsListener != nil {
            httpsListener.Close()
        }
    }()

    pa.server_name = ""
    if !parseArgs(&pa) {
        return 
    }

    if listenTcp(&httpListener, http_port) == false {
            goto EXIT
    }
    if listenTcp(&httpsListener, https_port) == false {
            goto EXIT
    }


    go installSign()
    for {
        select {
            case <- signChannel:
                fmt.Println("\nGet shutdown sign")
                //go notifyGoroutingExit()
                goto EXIT
        }
    }

    EXIT:
    fmt.Println("Waiting workers gorouting exit ....")
}

func installSign() {
    signal.Notify(signChannel, os.Interrupt, os.Kill)
}


func listenTcp(tcpListener **net.TCPListener, port int) (bool){
    local_host := "0.0.0.0:" + strconv.Itoa(port)
    tcpAddr, err := net.ResolveTCPAddr("tcp4", local_host)
    if err != nil {
        fmt.Printf("listenTcp, failed, local_host='%s', err=", local_host, err.Error())
        return false;
    }

    *tcpListener, err = net.ListenTCP("tcp", tcpAddr)
    if err != nil {
        panic("listenTcp, failed, err:" + err.Error())
        return false;
    }
    fmt.Printf("listenTcp, ok, listener=%d success on %s with tcp4.\n",  tcpListener, local_host)
    
    go handleAccept(*tcpListener, port)
    return true;
}


func handleAccept(listener *net.TCPListener, port int) {
    fmt.Printf("handleAccept, begin, port=%d.\n", port)
    for {
        client, err := listener.Accept()
        if err != nil {
            log.Panic(err)
        }

        fmt.Println("")
        fmt.Printf("handleAccept, client=%d, port=%d, comming...\n", client, port)
        go handleClientRequest(client, port)
    }
}

const (
    nXorLen = 128
)

func checkFileIsExist(filename string) bool {
    var exist = true
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        exist = false
    }
    return exist
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

type httpsClientHelloMsg struct {
   helloType                    byte
   vers                         [2]byte
   cententLen                   uint16//3bytes
   handShake                    byte
   handShakeLen                 uint16
   handShakeVersion             [2]byte
   random                       [32]byte
   sessionIdLen                 byte
   cipherSuitesLen              uint16
   compressionMethodsLen        byte
   extensionsLen                uint16
}

func byte2Int(data []byte) (int) {
       convInt := binary.BigEndian.Uint16(data)
       return (int)(convInt)
}

func getHostNameFromHttpsClientInfo(buf []byte) (string) {
    var clientHttps httpsClientHelloMsg 
    var hostname string
    
    fmt.Printf("getHostNameFromHttpsClientInfo, parse htts...\n")  
    i := 0
    clientHttps.helloType = buf[i]
    i += 1
    if  clientHttps.helloType != 22 {
        fmt.Printf("wrong helloType=%d, expect 22.", clientHttps.helloType)
        return "";
    }
 
    clientHttps.vers[0] = buf[i]
    clientHttps.vers[1] = buf[i+1]
    i += 2
    fmt.Printf("vers=%v\n", clientHttps.vers)
    if  clientHttps.helloType != 22 {
        fmt.Printf("wrong vers=%v, expect 3 1.", clientHttps.vers)
        return "";
    }

    clientHttps.cententLen = (uint16)(byte2Int(buf[i:i+2]))
    i += 3 
    fmt.Printf("helloType=%d, contentLen:=%x\n", clientHttps.helloType, clientHttps.cententLen)

    clientHttps.handShake = buf[i]
    i += 1
    clientHttps.handShakeLen = (uint16)(byte2Int(buf[i:i+2]))
    i += 2
    clientHttps.handShakeVersion[0] = buf[i]
    clientHttps.handShakeVersion[1] = buf[i+1]
    i += 2
    fmt.Printf("handshake=%x, len=%x, ver=%v\n", clientHttps.handShake, clientHttps.handShakeLen, clientHttps.handShakeVersion)

    i += 32;//skip random
 
    clientHttps.sessionIdLen = buf[i]
    i += 1
    i += (int)(clientHttps.sessionIdLen)
    fmt.Printf("sessionIdLen=%v\n", clientHttps.sessionIdLen)

    clientHttps.cipherSuitesLen = (uint16)(byte2Int(buf[i:i+2])) 
    i += 2
    i += (int)(clientHttps.cipherSuitesLen)
    fmt.Printf("cipherSuitesLen=%v\n", clientHttps.cipherSuitesLen)
    
    clientHttps.compressionMethodsLen = buf[i]
    i += 1
    i += (int)(clientHttps.compressionMethodsLen)
    fmt.Printf("compressionMethodsLen=%v\n", clientHttps.compressionMethodsLen)

    clientHttps.extensionsLen = (uint16)(byte2Int(buf[i:i+2]))
    i += 2
    fmt.Printf("extensionsLen: %d.\n", clientHttps.extensionsLen)
    
    var extensionType                [2]byte
    var extensionLen                 uint16

    for j := 0; j < (int)(clientHttps.extensionsLen); {
        
        extensionType[0] = buf[i]  
        extensionType[1] = buf[i+1] 
        i += 2
        j += 2
        extensionLen = (uint16)(byte2Int(buf[i:i+2]))
        i += 2
        j += 2
        fmt.Printf("    extenLen: %d.\n", extensionLen)
        fmt.Printf("    extenType: %v.\n", extensionType)

        if extensionType[0] == 0 && extensionType[1] == 0 {
             hostnameListLen := (uint16)(byte2Int(buf[i:i+2]))
             i += 2
             j += 2
             hostnameType := buf[i]
             i += 1
             j += 1
             hostnameLen := (uint16)(byte2Int(buf[i:i+2]))
             i += 2
             j += 2            
             hostname = (string)(buf[i:i+(int)(hostnameLen)])
             fmt.Printf("       hnListLen: %d, hnType=%x, hostname=%s.\n", hostnameListLen, hostnameType, hostname)
             break
        }
        i += (int)(extensionLen)
        j += (int)(extensionLen)
    }
 

    return hostname
}
//*/
func getHostNameFromHttpRequest(buf []byte) (string, string) {
    var hostName = ""
    var method = ""

    strRequest := string(buf)
    arrRequest := strings.Split(strRequest, "\r\n")
    if len(arrRequest) < 2 {
        fmt.Printf("getHostNameFromHttpRequest, len(arrRequest)=%d, data not enough.\n", len(arrRequest))
        return "", "";
    }
    
    arrMethod := strings.Split(arrRequest[0], " ")
    method = arrMethod[0]
    if method != "GET" && method != "CONNECT"  {
        fmt.Printf("getHostNameFromHttpRequest, arrRequest must begin with 'GET' or 'CONNECT', but '%s'.\n", arrRequest[0])
        return "", "";
    } 
    var foundHost = false
    for i := 1; i < len(arrRequest); i ++ {
        if strings.HasPrefix(arrRequest[i], "Host: ") {
            arrHost := strings.Split(arrRequest[i], " ")
            hostName = arrHost[1]
            foundHost = true
            break
        } 
    }
    if !foundHost {
        fmt.Printf("executeTask, not found 'Host: ' in arrRequest.\n")
        return "", "";
    } 
    return hostName, method
}

func handleClientRequest(client net.Conn, serverPort int) {
	if client == nil {
		return
	}
        
        bIsClient := true
        if pa.server_name == "" {
            bIsClient = false
        }
        //key 
        key := make([]byte, 8)
        key[0] = 6
        key[1] = 7
        key[2] = 1
        key[3] = 3
        key[4] = 2
        key[5] = 5
        key[6] = 4
        key[7] = 0

        var proxy net.Conn = nil
        defer func() {
            fmt.Printf("handleClientRequest, closed client=%d and proxy=%d.\n", client, proxy)	
            client.Close()
            if proxy != nil {
                proxy.Close()
            }
        }()

        b := make([]byte, 8192)
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}

        if !bIsClient {
            //decode data
            fmt.Printf("handleClientRequest, server recv encode data[0-4]: '%s', ", b[0:4])	
            xorCodec(b, key, nXorLen)            
            fmt.Printf("decode data[0-4]: '%s'.\n", b[0:4])	
        } 

        var hostname, method string
        hostname, method = getHostNameFromHttpRequest(b[0:n]) //https maybe contains CONNECT cmd.
        if hostname == "" {
            if serverPort == 443 {
                hostname = getHostNameFromHttpsClientInfo(b[0:n])
                if hostname == "" {
                   fmt.Printf("handleClientRequest, no hostname found by getHostNameFromHttpsClientInfo.\n")
                    return
                }
            } else {
                fmt.Printf("handleClientRequest, no hostname found by getHostNameFromHttpRequest.\n")
                return
            }
        } 
        fmt.Printf("handleClientRequest, got hostname: '%s'.\n", hostname)	
	var address string
        if !bIsClient {
            arrHostName := strings.Split(hostname, ":")//sometime hostname's format is 'hostname:port'
            if len(arrHostName) == 1 {
                address = hostname + ":" +  strconv.Itoa(serverPort)
            } else if len(arrHostName) == 2 { 
                address = hostname
            } else {
                fmt.Printf("handleClientRequest, wrong format of hostname='%s'.\n", hostname)
                return
            }
        } else {
            address = pa.server_name + ":" +  strconv.Itoa(serverPort)
        }
	
        //proxy data 
	proxy, err = net.Dial("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}
        fmt.Printf("handleClientRequest, client=%d, dial to address='%s' ok, proxy=%d.\n", client, address, proxy)
        fmt.Printf("handleClientRequest, client recv data=\n'%s'\n", b[0:n])	
	
        //encrypt data    
        if bIsClient {//client
            fmt.Printf("handleClientRequest, client send data[0-4]: '%s', ", b[0:4])	
            xorCodec(b, key, nXorLen)            
            fmt.Printf("encode data[0-4]: '%s'.\n", b[0:4])	
        } else {
            fmt.Printf("handleClientRequest, server send data[0-4]: '%s'.\n", b[0:4])	
        }

	if method == "CONNECT" {
            if bIsClient {
	    	fmt.Fprint(client, "handleClientRequest, HTTP/1.1 200 Connection established\r\n\r\n")
		fmt.Println("handleClientRequest, client connect cmd, response directly, prxoy data to server.")
		proxy.Write(b[:n])
            }  
	} else {
		proxy.Write(b[:n])
		fmt.Println("handleClientRequest, other cmd, proxy data...")
	}

/*
        proxy.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
        for {
            n, err := proxy.Read(b[:])
            if err != nil {
                errString := err.Error()
                fmt.Printf("handleClientRequest, proxy read connection error:%s.\n", errString)
                switch {
                case strings.Contains(errString, "timeout"):
                    break;
                default:
                    break;
                }
                break;
            }

            ret, err := client.Write(b[0:n])
            if err != nil {
                errString := err.Error()
                fmt.Println("handleClientRequest, client write error: " + errString)
            }
            if ret != n {
                fmt.Printf("handleClientRequest, client write ret=%d, but n=%d.\n", ret, n)              
            }           
        }
//*/   
 	go io.Copy(proxy, client)
	io.Copy(client, proxy)
        fmt.Printf("handleClientRequest, client=%d, proxy=%d, proxy data end\n", client, proxy)
}


func xorCodec(data []byte, key []byte, lenCodec int) {
    //return 
    n := len(data)
    if n > lenCodec {
        n = lenCodec
    }
    nKeyLen := len(key)
    
    for i:=0; i<n; i++ {
        data[i] ^= key[i%nKeyLen]
    } 
}


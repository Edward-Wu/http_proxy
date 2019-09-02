# http_proxy
a simple proxy vpn tool with golang language which support http and https.

#Compile both on server and client
$ go build http_proxy.go

#run on server, which must be able to access the domainname than you want to. run it maybe need sudo.
$ sudo ./http_proxy

#config on client
1.in hosts file (/etc/hosts), add any domainname that you can't access because of net limit or check(you know why) to map to 
$ vi /etc/hosts
local ip (127.0.0.1), such as :
127.0.0.1 www.google.com

#run on client
$ sudo ./http_proxy -s your.server.hostname(ip/domain name)

#test on client with net tool or browser
$ curl www.google.com

if you get the response content of www.google.com , congradulation! 

#issue
1.on mac, you can use safari browser not chrome browser, I will analyse why chrome browser can't work well in future.



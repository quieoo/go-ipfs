#!/usr/bin/expect -f

set user [lindex $argv 0]

set addr [lindex $argv 1]

set passwd [lindex $argv 2]

set timeout -1

spawn scp ipfs origin pbitswap pbitswap-static $user@$addr://home/IPFS/

expect {
"*password:" { send "$passwd\n" }
}
expect eof
exit


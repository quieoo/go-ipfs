#!/bin/bash



#targets=('embed78' 'embed67' 'embed66' 'embed79' 'embed61' 'embed62' 'embed64' 'embed65')
addrs=('47.243.113.177' '161.117.252.108' '47.74.64.69' '147.139.4.84' '147.139.169.90' '47.74.16.9' '8.208.99.163' '47.91.124.16')

length=${#addrs[@]}

for ((k=0;k<$length;k++))
do
 expect expectmain.sh 'root' ${addrs[$k]} '2'
done


if [ ! -n "$1" ];then
    echo "panic : must input the password for keystores"
    exit
fi
if [ ! -n "$2" ];then
    echo "panic : must input the key for REQSMPCADDR"
    exit
fi
rpc="http://127.0.0.1:5871"
./gsmpc-client -cmd ACCEPTREQADDR  -url $rpc --keystore ./keystores/smpc_k --passwd $1 -key $2 --keytype ED25519 -mode 0


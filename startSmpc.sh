rpcport=5871
port=48541
boot="enode://c9774dfbcc77dc3f6c849e7ab972a2bed3e3e6304364aa9aafb0a830c1f12e55bc34f3da428e3e1ffb4e3f57fccb4413fe32fb71005657882a6e2f510e660825@18.162.150.38:4440"
result=`go version`
if [[ $result == "" ]] ; then
    exit
fi
result=`make -v`
if [[ $result == "" ]] ; then
    exit
fi
if [ ! -n "$1" ];then
    echo "panic : must input the password for keystores"
    exit
fi
if [ ! -d "./logs" ];then
    mkdir logs
fi
if [ ! -d "./keystores" ];then
    mkdir keystores
fi
if [ ! -f "./keystores/smpc_k" ];then
    rm -rf ./biota-swap
    git clone https://github.com/TanglePay/biota-swap
    cd biota-swap
    go build -ldflags "-w -s"
    ./bwrap -key $1 ../keystores/smpc_k
    if [ ! -f "./keystores/evm_k" ];then
        ./bwrap -key $1 ../keystores/evm_k
    fi    
    cd ..
fi
if [ ! -f "./gsmpc" ] || [ ! -f "./gsmpc-client" ];then
    rm -rf ./smpc-node
    git clone https://github.com/TanglePay/smpc-node
    cd smpc-node && make all
    cp ./build/bin/gsmpc ..
    cp ./build/bin/gsmpc-client ..
    cd ..
fi

result=`echo -e "\n" | telnet 127.0.0.1 $rpcport 2> /dev/null | grep Connected | wc -l`
if [ $result -ne 1 ]; then
    nohup ./gsmpc --rpcport $rpcport --bootnodes $boot --port $port --nodekey "./logs/node.key" --verbosity 3 > ./logs/node.log 2>&1 &
    echo "wait for a few seconds to start gsmpc service."
    for i in {1..300}
    do 
        result=`echo -e "\n" | telnet 127.0.0.1 $rpcport 2> /dev/null | grep Connected | wc -l`
        if [ $result -eq 1 ]; then
            echo "$i"
            break
        else 
            echo -ne "$i\r"
        fi
        sleep 1
    done
fi

./gsmpc-client -cmd EnodeSig -url http://127.0.0.1:$rpcport --keystore ./keystores/smpc_k --passwd $1

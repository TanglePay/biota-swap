echo -n "Enter database's password:"
read -s pwd
echo ""
if [ -z "$pwd" ];then
    echo -e "\e[31m panic!!! Database's password cann't be empty"
    exit
fi

result=`go version`
if [[ $result == "" ]] ; then
    echo -e "\e[31m !!! panic : golang is not installed"
    exit
fi

pkill bwrap_main
if [ ! -f "./bwrap_main" ]; then
    rm -rf ./biota-swap
    git clone https://github.com/TanglePay/biota-swap
    cd biota-swap
    go build -ldflags "-w -s"
    cp bwrap ../bwrap_main
    cd ..
fi

if [ ! -d "./config" ];then
    mkdir config
fi

if [ -f "./smpc_k" ];then
    mv smpc_k ./config/
fi

if [ ! -f "./config/smpc_k" ];then
    echo -e "\e[31m !!! panic : Must cp the smpc_k file to the path of ./config/"
    exit
fi

SmrRpcUrl="https://json-rpc.evm.shimmer.network"
SmrWssUrl="wss://ws.json-rpc.evm.shimmer.network"
IotaRpcUrl="https://chrysalis-nodes.iota.org"
IotaWssUrl="wss://chrysalis-nodes.iota.org/mqtt"

tanglePay=$(echo "0xfb6e712F4f71D418A298EBe239889A2496f1359b" | tr '[:upper:]' '[:lower:]')
soonavers=$(echo "0x3Fdd4B2d69848F74E44765e6AD423198bdBD94fa" | tr '[:upper:]' '[:lower:]')
tangleswa=$(echo "0x380dF538Ab2587B11466d07ca5c671d33497d5Ca" | tr '[:upper:]' '[:lower:]')
dltgreenp=$(echo "0x5e80cf0C104D2D4f685A15deb65A319e95dd80dD" | tr '[:upper:]' '[:lower:]')
spyce5xxx=$(echo "0x9dcb974Cf7522F91F2Add8303e7BCB2221063c48" | tr '[:upper:]' '[:lower:]')
govtreasu=$(echo "0xeBbe638eF6dF4A3837435bB44527f8D9BA9CF981" | tr '[:upper:]' '[:lower:]')

account_fill=$(<config/smpc_k)
addr=${account_fill: 12: 40}
addr="0x"$addr
if [ "$addr" == "$tanglePay" ]; then
    SubGroup="20b5e962f662e74eca9e63596c57bfce2b160aded631b3b613148ebd47696cfe710a2128de153c48d17e29fdf11230ba40d04b626dd32b2af6fcf5e93a0cc52e"
    EthRpcUrl="https://mainnet.infura.io/v3/3f8b4373a4a943bf8b9c635fba90ee78"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/3f8b4373a4a943bf8b9c635fba90ee78"
elif [ "$addr" == "$soonavers" ]; then
    SubGroup="025c6714bad166f47a16a1da5e0b34d4cd4c3b0eddec31ca0bcaa9f89ac0d29e997026f5d66e17c01c035eabe8e04ac258a65363438904788a24a37ae3bcd97a"
    EthRpcUrl="https://mainnet.infura.io/v3/f3d3066e39d7480298e3b921927dd234"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/f3d3066e39d7480298e3b921927dd234"
elif [ "$addr" == "$tangleswa" ]; then
    SubGroup="9f3479126e84944121add787f8ad1e8145c53d74cbfee78ffeffd7e05cadab8c94d5cb21c6ffde7af9697e10c390f5c86ecfa69efae1dee85e829bda9428ee35"
    EthRpcUrl="https://mainnet.infura.io/v3/3640e819dfa3470092c453ccdbf506a7"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/3640e819dfa3470092c453ccdbf506a7"
elif [ "$addr" == "$dltgreenp" ]; then
    SubGroup="c1b744444161eca1f1284d869f9b9b24a3597934f069863ac484ceb2ef271e4f1d8154628ac43be9754bb11e98f2bd851b6fdc55a3b05681d9c8ba901e4ea17b"
    EthRpcUrl="https://mainnet.infura.io/v3/796c3600d73a4a3c99be992f1f1035c7"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/796c3600d73a4a3c99be992f1f1035c7"
elif [ "$addr" == "$spyce5xxx" ]; then
    SubGroup="ed848bf18a6f2d968179b3f175bcd1fa87691b83cefb04722982e77c0d1292ae0fcfd3816c992b0ea53ea60e882e75a8ec60ee9c27b98622d25029acbdc44ecf"
    EthRpcUrl="https://mainnet.infura.io/v3/6ea7d0e6c4304751b2060044f2b213bd"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/6ea7d0e6c4304751b2060044f2b213bd"
elif [ "$addr" == "$govtreasu" ]; then
    SubGroup="81fc498fbc4d52b741cc55be8eb814ae0d0dc24bbcbf481e15dd9703c823208c382d655f93a53236f6ce24bfa7ed77d2411b4323229e846690181537a0ef24a6"
    EthRpcUrl="https://mainnet.infura.io/v3/0e0a929c16b947199c9290661c320ca6"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/0e0a929c16b947199c9290661c320ca6"
else
    echo -e "\e[31m !!! panic : address error. $addr is not in the Group"
    exit
fi

cat> config/conf.toml << :EOF
# NodeUrl is the smpc node rpc url
# Gid is the subgroup id
# ThresHold is the group rule. It can be "2/3", "3/5", "4/6" ...
# KeyStore is the wallet account of the node of smpc
[Smpc]
NodeUrl = "http://127.0.0.1:5871"
Gid = "$SubGroup"
ThresHold   ="4/6"
KeyStore = "./config/smpc_k"

# The Server config
# DetectCount is the detect count when it request a sign to accept. The DetectTime is the time as seconds between two detect loops.
# AcceptTime is the check time as seconds with one loop.
# AcceptOverTime is the time as seconds. If smpc sign over this time, it should be not accepted.
[Server]
DetectCount = 60
DetectTime = 10
AcceptTime = 30
AcceptOverTime = 7200

# database driver is mysql 46768bacc61d97fe9d459fcb01181dcb6fae36f9
# the dabasebase name is "smpc" and the table to see the "readme"
[Db]
Host = "18.162.150.38"
Port = "3306"
DbName = "smpc_main"
Usr= "smpcNode"
Pwd= "$pwd"

[TxErrorRecord]
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
Contract = "0x3C71B92D6f54473a6c66010dF5Aa139cD42c34b0"
ScanEventType = 0
TimePeriod = 3600

# Tokens contain "ATOI", "IOTA", SMIOTA", "MATIC"
# Symbol is the unique
# ScanEventType, 0: listen event as websockt or mqtt; 1: scan block to get event logs.
# MultiSignType, 0 is contract multiSign, 2 is smpc multiSign
# MultiSignType = 0: PublicKey is null
# MultiSignType = 2: Contract and KeyStore is null
[[Tokens]]
Symbol = "ETH"
NodeRpc = "$EthRpcUrl"
NodeWss = "$EthWssUrl"
ScanEventType = 0
ScanMaxHeight = 10000
MultiSignType = 0
Contract = "0x7C32097EB6bA75Dc5eF370BEC9019FD09D96ab9d"
MinAmount = 1
KeyStore = "./config/smpc_k"
GasPriceUpper = 40

[[Tokens]]
Symbol = "WBTC"
NodeRpc = "$EthRpcUrl"
NodeWss = "$EthWssUrl"
ScanEventType = 0
ScanMaxHeight = 10000
MultiSignType = 0
Contract = "0x6c2F73072bD9bc9052D99983e36411f48fa6cDf0"
MinAmount = 1
KeyStore = "./config/smpc_k"
GasPriceUpper = 40

[[Tokens]]
Symbol = "sETH"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
ScanMaxHeight = 1000
MultiSignType = 0
Contract = "0xa158A39d00C79019A01A6E86c56E96C461334Eb0"
MinAmount = 1
KeyStore = "./config/smpc_k"
GasPriceUpper = 10

[[Tokens]]
Symbol = "sBTC"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
ScanMaxHeight = 1000
MultiSignType = 0
Contract = "0x1cDF3F46DbF8Cf099D218cF96A769cea82F75316"
MinAmount = 1
KeyStore = "./config/smpc_k"
GasPriceUpper = 10

# Pairs is the bridge pair. 
# SrcToken to DestToken. They must be in the "Tokens".
[[Pairs]]
SrcToken = "ETH"
DestToken = "sETH"

[[Pairs]]
SrcToken = "WBTC"
DestToken = "sBTC"
:EOF

./bwrap_main -d
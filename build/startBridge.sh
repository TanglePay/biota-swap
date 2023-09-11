echo -n "Enter database's password:"
read -s pwd
echo ""
if [ -z "$pwd" ];then
    echo -e "\e[31m panic!!! Database's password cann't be empty"
    exit
fi

pkill bswap
result=`go version`
if [[ $result == "" ]] ; then
    echo -e "\e[31m !!! panic : golang is not installed"
    exit
fi

if [ ! -f "./bwrap" ]; then
    rm -rf ./biota-swap
    git clone https://github.com/TanglePay/biota-swap
    cd biota-swap
    go build -ldflags "-w -s"
    cp bwrap ../
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

if [ $1 = "test" ]; then
    SmrRpcUrl="https://json-rpc.evm.testnet.shimmer.network/"
    SmrWssUrl="wss://ws.json-rpc.evm.testnet.shimmer.network/"
    IotaRpcUrl="https://api.lb-0.h.chrysalis-devnet.iota.cafe"
    IotaWssUrl="wss://api.lb-0.h.chrysalis-devnet.iota.cafe/mqtt"
else
    SmrRpcUrl="https://json-rpc.evm.mainnet.shimmer.network/"
    SmrWssUrl="wss://ws.json-rpc.evm.mainnet.shimmer.network/"
    IotaRpcUrl="https://chrysalis-nodes.iota.org"
    IotaWssUrl="wss://chrysalis-nodes.iota.org/mqtt"
fi

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
    EthRpcUrl="https://polygon-mumbai.g.alchemy.com/v2/fXcphe9VsC6eWwEjRWaSz2ATxI_Sxn0K"
    EthWssUrl="wss://polygon-mumbai.g.alchemy.com/v2/fXcphe9VsC6eWwEjRWaSz2ATxI_Sxn0K"
elif [ "$addr" == "$soonavers" ]; then
    SubGroup="025c6714bad166f47a16a1da5e0b34d4cd4c3b0eddec31ca0bcaa9f89ac0d29e997026f5d66e17c01c035eabe8e04ac258a65363438904788a24a37ae3bcd97a"
    EthRpcUrl="https://rough-flashy-scion.matic-testnet.discover.quiknode.pro/4c2aa9a5803d25233442aa9eed9a108de0d85503/"
    EthWssUrl="wss://rough-flashy-scion.matic-testnet.discover.quiknode.pro/4c2aa9a5803d25233442aa9eed9a108de0d85503/"
elif [ "$addr" == "$tangleswa" ]; then
    SubGroup="9f3479126e84944121add787f8ad1e8145c53d74cbfee78ffeffd7e05cadab8c94d5cb21c6ffde7af9697e10c390f5c86ecfa69efae1dee85e829bda9428ee35"
    EthRpcUrl="https://muddy-chaotic-bridge.matic-testnet.discover.quiknode.pro/cc49cd5cf13bc79b1c2bb9b771aa7447d021741e/"
    EthWssUrl="wss://muddy-chaotic-bridge.matic-testnet.discover.quiknode.pro/cc49cd5cf13bc79b1c2bb9b771aa7447d021741e/"
elif [ "$addr" == "$dltgreenp" ]; then
    SubGroup="c1b744444161eca1f1284d869f9b9b24a3597934f069863ac484ceb2ef271e4f1d8154628ac43be9754bb11e98f2bd851b6fdc55a3b05681d9c8ba901e4ea17b"
    EthRpcUrl="https://wiser-proportionate-meme.matic-testnet.discover.quiknode.pro/c7912538114a5e92343154802e04960ab605d05b/"
    EthWssUrl="wss://wiser-proportionate-meme.matic-testnet.discover.quiknode.pro/c7912538114a5e92343154802e04960ab605d05b/"
elif [ "$addr" == "$spyce5xxx" ]; then
    SubGroup="ed848bf18a6f2d968179b3f175bcd1fa87691b83cefb04722982e77c0d1292ae0fcfd3816c992b0ea53ea60e882e75a8ec60ee9c27b98622d25029acbdc44ecf"
    EthRpcUrl="https://restless-smart-night.matic-testnet.discover.quiknode.pro/8977486437b86d8f91d8848e3206fa7958b42aca/"
    EthWssUrl="wss://restless-smart-night.matic-testnet.discover.quiknode.pro/8977486437b86d8f91d8848e3206fa7958b42aca/"
elif [ "$addr" == "$govtreasu" ]; then
    SubGroup="81fc498fbc4d52b741cc55be8eb814ae0d0dc24bbcbf481e15dd9703c823208c382d655f93a53236f6ce24bfa7ed77d2411b4323229e846690181537a0ef24a6"
    EthRpcUrl="https://polygon-mumbai.g.alchemy.com/v2/WsQG9XqmykaRSiHi-aO5szCcnDU_4Nsd"
    EthWssUrl="wss://polygon-mumbai.g.alchemy.com/v2/WsQG9XqmykaRSiHi-aO5szCcnDU_4Nsd"
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
DbName = "smpc"
Usr= "smpcNode"
Pwd= "$pwd"

[TxErrorRecord]
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
Contract = "0xac1A077b6F9f52Bd225d3E43Bc4EdBb7F464bA31"
ScanEventType = 0
TimePeriod = 3600

# Tokens contain "ATOI", "IOTA", SMIOTA", "MATIC"
# Symbol is the unique
# ScanEventType, 0: listen event as websockt or mqtt; 1: scan block to get event logs.
# MultiSignType, 0 is contract multiSign, 2 is smpc multiSign
# MultiSignType = 0: PublicKey is null
# MultiSignType = 2: Contract and KeyStore is null
[[Tokens]]
Symbol = "ATOI"
NodeRpc = "$IotaRpcUrl"
NodeWss = "$IotaWssUrl"
ScanEventType = 0
MultiSignType = 2
PublicKey = "b477a4b11a54a6a1a3aa792878f50b49e21536bf0bfdd0876ec99fae4e4bdb08"
MinAmount = 1000000

[[Tokens]]
Symbol = "MATIC"
NodeRpc = "$EthRpcUrl"
NodeWss = "$EthWssUrl"
ScanEventType = 0
ScanMaxHeight = 10000
MultiSignType = 0
Contract = "0x0F326747787Ec6894e9be2Af691d7451ea596660"
MinAmount = 1
KeyStore = "./config/smpc_k"

[[Tokens]]
Symbol = "WBTC"
NodeRpc = "$EthRpcUrl"
NodeWss = "$EthWssUrl"
ScanEventType = 0
ScanMaxHeight = 10000
MultiSignType = 0
Contract = "0xBF2F528b7d6b30Ede64a7a2D6C4819802831551b"
MinAmount = 1
KeyStore = "./config/smpc_k"

[[Tokens]]
Symbol = "sMIOTA"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
ScanMaxHeight = 1000
MultiSignType = 0
Contract = "0x1f7A07357e1E3e74A564fDeE8030f06F296AD540"
MinAmount = 1000000
KeyStore = "./config/smpc_k"

[[Tokens]]
Symbol = "sMATIC"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
ScanMaxHeight = 1000
MultiSignType = 0
Contract = "0xB6D0FBe198bf48c7E7dad8b457994a0dBac795Ef"
MinAmount = 1
KeyStore = "./config/smpc_k"

[[Tokens]]
Symbol = "sWTBC"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
ScanMaxHeight = 1000
MultiSignType = 0
Contract = "0x69499Cf6b0244AD7CEA28F6eeeb4EF4d8cd1Bc33"
MinAmount = 1
KeyStore = "./config/smpc_k"


# Pairs is the bridge pair. 
# SrcToken to DestToken. They must be in the "Tokens".
[[Pairs]]
SrcToken = "ATOI"
DestToken = "sMIOTA"

[[Pairs]]
SrcToken = "MATIC"
DestToken = "sMATIC"

[[Pairs]]
SrcToken = "WBTC"
DestToken = "sWBTC"
:EOF

./bwrap -d
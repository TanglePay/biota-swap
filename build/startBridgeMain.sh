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
rm bwrap_main
rm -rf ./biota-swap
git clone https://github.com/TanglePay/biota-swap
cd biota-swap
go build -ldflags "-w -s"
cp bwrap ../bwrap_main
cd ..

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
IotaRpcUrl="https://api.stardust-mainnet.iotaledger.net"
IotaWssUrl="https://api.stardust-mainnet.iotaledger.net"

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
    SubGroup="dd235cc35050963bedc23d8a78c4fb8488822f870c914f0814549d4018e81aaab1c945376e71ab0bbdd822d456729141e31c3f232712b3df45b72e552c369eda"
    EthRpcUrl="https://mainnet.infura.io/v3/3f8b4373a4a943bf8b9c635fba90ee78"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/3f8b4373a4a943bf8b9c635fba90ee78"
elif [ "$addr" == "$soonavers" ]; then
    SubGroup="01cb75848e61c9db2b07a1f33424dac05b29203799198edbb644ba90459ac8332d4277363ddb93b8c84baa2ca857f1753b00910b84f764da050a69a7ac4ca171"
    EthRpcUrl="https://mainnet.infura.io/v3/f3d3066e39d7480298e3b921927dd234"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/f3d3066e39d7480298e3b921927dd234"
elif [ "$addr" == "$tangleswa" ]; then
    SubGroup="ddd7cf8f59f097d2dc71e6cc654fa14e33b667548ec3bb9bd635c25008f5ab01b628a32ae063cb24dac680cf538935adf2f2dffaa117102c20c4a3726b3b62b1"
    EthRpcUrl="https://mainnet.infura.io/v3/3640e819dfa3470092c453ccdbf506a7"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/3640e819dfa3470092c453ccdbf506a7"
elif [ "$addr" == "$dltgreenp" ]; then
    SubGroup="f83162f70c1803f4858bedb545508c68e1611cbbbf03bcfdf295d09c8ec981a98d689dc5551d01e539e290a58b50f9d4cd039243d581113530b78a3afedefb1f"
    EthRpcUrl="https://mainnet.infura.io/v3/796c3600d73a4a3c99be992f1f1035c7"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/796c3600d73a4a3c99be992f1f1035c7"
elif [ "$addr" == "$spyce5xxx" ]; then
    SubGroup="ed848bf18a6f2d968179b3f175bcd1fa87691b83cefb04722982e77c0d1292ae0fcfd3816c992b0ea53ea60e882e75a8ec60ee9c27b98622d25029acbdc44ecf"
    EthRpcUrl="https://mainnet.infura.io/v3/6ea7d0e6c4304751b2060044f2b213bd"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/6ea7d0e6c4304751b2060044f2b213bd"
elif [ "$addr" == "$govtreasu" ]; then
    SubGroup="db9af9dd883adef5a348ac365dd9d4de53bcf417b09ed03934af8a9d2f189d74d71603e4eb6f21540b2dea016a0cd3451fb401aa656e6818a76536c904f639aa"
    EthRpcUrl="https://mainnet.infura.io/v3/0e0a929c16b947199c9290661c320ca6"
    EthWssUrl="wss://mainnet.infura.io/ws/v3/0e0a929c16b947199c9290661c320ca6"
else
    echo -e "\e[31m !!! panic : address error. $addr is not in the Group"
    exit
fi

cat> config/conf.toml << :EOF
# Version
Version = "1.0.1"
# PendingTime is time of seconds for a tx keep pending status
PendingTime = 300

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
AcceptOverTime = 72000

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
Contract = "0xD9B13709Ce4Ef82402c091f3fc8A93a9360A5c1e"
ScanEventType = 0
TimePeriod = 3600

# Tokens contain "IOTA", "ETH", WBTC", "sIOTA", "sETH", "sBTC"
# Symbol is the unique
# ScanEventType, 0: listen event as websockt or mqtt; 1: scan block to get event logs.
# MultiSignType, 0 is contract multiSign, 2 is smpc multiSign
# MultiSignType = 0: PublicKey is null
# MultiSignType = 2: Contract and KeyStore is null
[[Tokens]]
Symbol = "IOTA"
NodeRpc = "$IotaRpcUrl"
NodeWss = "$IotaWssUrl"
ScanEventType = 0
MultiSignType = 2
# iota1qr3jf395mx0frslvndkzkhwe63gvwwqynh7997xm46h2lk6gv78dg5n27nc
PublicKey = "1bcd460eb168c5de3183eca59c9b960f8083fdd703aec23df6a2815bffac0254"
MinAmount = 1

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
Symbol = "sIOTA"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
ScanMaxHeight = 1000
MultiSignType = 0
Contract = "0x5dA63f4456A56a0c5Cb0B2104a3610D5CA3d48E8"
MinAmount = 1
KeyStore = "./config/smpc_k"
GasPriceUpper = 0

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
GasPriceUpper = 0

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
GasPriceUpper = 0

# Pairs is the bridge pair. 
# SrcToken to DestToken. They must be in the "Tokens".
[[Pairs]]
SrcToken = "IOTA"
DestToken = "sIOTA"

[[Pairs]]
SrcToken = "ETH"
DestToken = "sETH"

[[Pairs]]
SrcToken = "WBTC"
DestToken = "sBTC"
:EOF

./bwrap_main
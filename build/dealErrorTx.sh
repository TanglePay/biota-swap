rm -rf ./biota-swap
git clone https://github.com/TanglePay/biota-swap
cd biota-swap
git checkout error_deal
go build -ldflags "-w -s"
cp bwrap ../bdeal_err
cd ..

if [ ! -f "./config/smpc_k" ];then
    echo -e "\e[31m !!! panic : Must cp the smpc_k file to the path of ./config/"
    exit
fi

./bdeal_err ETH sETH 0x253447775ff379c6f72bbc2b4511a34ff9a50aa8636ad85e6e8eeee4ac31f494
./bdeal_err WBTC sBTC 0x2b33595fd3d1c5a698d4a9042b3fc120d23c9ce1228e8c0ee43ff2d4ce4e4174
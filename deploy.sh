make all
go build -o reraffle-linux
upx reraffle-linux
rsync -avzrP reraffle-linux god@89.223.25.141:/home/god/bot/reraffle-linux
ssh god@89.223.25.141 "sudo systemctl stop bot.service && sudo systemctl start bot.service"

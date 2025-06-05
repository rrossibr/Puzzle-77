After downloading, use go mod tidy for updates.

build on Mac with: go build -o puzzle77

run with: ./puzzle77 -cores * 
Change * for number of cores that you like to use!

Open file puzzle77.go

By default the program is set for bitcoin wallet #71

var (
	startHex      = "400000000000000000"
	endHex        = "7fffffffffffffffff"
	targetAddress = "1PWo3JeB9jrGwfHDNpdGK54CRas7fsVzXU"
)

Change startHex, endHex and TargetAddress

Use only to try to find Bitcoin Puzzle Wallets!

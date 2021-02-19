module github.com/solarwinds/snap-plugin-lib/v2/tutorial

go 1.13

require (
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/shirou/gopsutil v2.20.9+incompatible
	github.com/solarwinds/snap-plugin-lib/v2 v2.0.4
	github.com/stretchr/testify v1.6.1
)

replace github.com/solarwinds/snap-plugin-lib/v2 => ./..

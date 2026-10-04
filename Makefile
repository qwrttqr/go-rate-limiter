.PHONY: bench bench-race

PKG     ?= ./bench

COUNT   ?= 10
CPUS    ?= 1,4,8,12


bench:
	go test -run='^$$' -bench=. -benchmem -count=$(COUNT) -cpu=$(CPUS) $(PKG)

bench-race:
	go test -run='^$$' -bench=. -race $(PKG)


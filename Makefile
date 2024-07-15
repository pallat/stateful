.Phony: clean

clean:
	rm -rf ./postgres/mount/*
load:
	cat api.http | vegeta attack -duration=5s -rate=40/s | tee results.bin | vegeta report
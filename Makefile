.PHONY: all clean

all:
	docker build -t prohibit-paths-plugin .
	mkdir -p plugin/rootfs/run/docker/plugins
	docker run --rm prohibit-paths-plugin \
		cat docker-mountpoint-prohibit-paths \
		> plugin/rootfs/docker-mountpoint-prohibit-paths
	chmod +x plugin/rootfs/docker-mountpoint-prohibit-paths
	cp config.json plugin/
	docker plugin create prohibit-paths plugin

clean:
	docker rmi prohibit-paths-plugin || true
	docker plugin rm prohibit-paths || true
	rm -rf plugin

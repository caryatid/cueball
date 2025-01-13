package vm

import (
	"context"
	"fmt"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/worker"
	"text/template"
	"github.com/caryatid/cueball/retry"
	"net"
	"net/url"
	"net/http"
	"strings"
)


var (
	binaries = []string{"vmlinuz-virt", "initramfs-virt", "modloop-virt"}
	sep = ":"

	up = `
if -n { eltest -e /sys/class/net/{{.Bridge.Name}} }
if { ip link add name {{.Bridge.Name}} type bridge }
if { ip addr add {{.Gateway.String()/${cidr} dev {{.Wan.Name}} }
if { ip link set dev {{.Wan.Name}} up }
pipeline { echo \"
destroy table inet {{.Wan.Name}}.wan
destroy table ip {{.Wan.Name}}.nat
table inet {{.Wan.Name}}.wan {
	chain output {
		type filter hook output priority 100; policy accept;
	}
	chain input {
		type filter hook input priority 0; policy accept;
		#iifname {{.Wan.Name}} accept
		#iifname ${wan} accept
	}
	chain forward {
		type filter hook forward priority 0; policy accept;
		iifname ${bridge} oifname ${wan} accept
		# iifname ${wan} oifname ${bridge} ct state related,established accept
		iifname ${wan} oifname ${bridge} accept
	}
}
table ip ${bridge}.nat {
	chain postrouting {
		type nat hook postrouting priority 100; policy accept;
		iifname ${bridge} oifname ${wan} masquerade
	}
}
`

	http_conf = `
server.document-root=\"${platoon}\"
server.port = ${http_port}
server.bind = \"${gw}\"
dir-listing.activate = \"enable\"" 
}
if { redirfd -w 1 type echo longrun }
redirfd -w 1 run echo "#!/bin/execlineb -P\nlighttpd -D -f ./data/${squad}-http.conf
`

	http_run =`
#!/bin/execlineb -P
lighttpd -D -f ./data/${squad}-http.conf

`

	dns_conf = `
auth-server=${squad},${bridge}
auth-soa=${create_date},${squad},1200,120,604800
auth-zone=${squad},${bridge}
interface=${bridge}
bind-interfaces
except-interface=lo
domain=${squad}
bogus-priv
# domain-needed
no-resolv
no-hosts
dhcp-leasefile=${platoon}/state/${squad}-dns.leases
dhcp-hostsfile=${platoon}/state/${squad}-dns.hosts
log-queries
dhcp-option=option:router,${gw}
dhcp-option=option:dns-server,${upstream}
dhcp-range=${dhs},${dhe}
dhcp-authoritative
host-record=${squad},${gw}
#servers-file=${platoon}/state/${squad}-dns.servers
#server=/${squad}/${gw}
server=${upstream}
`

	dns_run = `
#!/bin/execlineb -P
dnsmasq -d --user=root --conf-file=./data/${squad}-dns.conf"
`

	vm_run = `
"#!/bin/execlineb -s1
define host \\${1}-${team}.${squad}
backtick -E tap { qc-tap }
backtick -E mac { qc-net-cvt str2mac ${host} } 
foreground {
	sh -c 
\"tf=$(mktemp); touch ${platoon}/state/${squad}-dns.hosts; cp ${platoon}/state/${squad}-dns.hosts $tf
echo ${mac},${host} | awk -F, 's[$1] { next } { print; s[$1]=1 }' - $tf >${platoon}/state/${squad}-dns.hosts\"
}
foreground { sudo s6-svc -h ${platoon}/scan/${squad}-dns }
foreground { ip link set dev ${tap} master ${bridge} }
foreground { ip link set dev ${tap} up }
ifelse { test -e ${platoon}/9p/cache/lbu/${team}.${squad}.apkovl.tar.gz } {
	qemu-system-x86_64 -smp cpus=${cpus} -cpu host -m ${mem} -device virtio-rng-pci
	-kernel ${platoon}/boot/vmlinuz-virt -initrd ${platoon}/boot/initramfs-virt -enable-kvm
	-append \"ip=dhcp modloop=http://${url}/boot/modloop-virt autodetect_serial=no apkovl=http://${url}/9p/cache/lbu/${team}.${squad}.apkovl.tar.gz\"
	-nic tap,model=virtio-net-pci,ifname=${tap},script=no,downscript=no,mac=${mac} 
	${9pargs}  -display none
}
qemu-system-x86_64 -smp cpus=${cpus} -cpu host -m ${mem} -device virtio-rng-pci
-kernel ${platoon}/boot/vmlinuz-virt -initrd ${platoon}/boot/initramfs-virt -enable-kvm
-append \"ip=dhcp modloop=http://${url}/boot/modloop-virt autodetect_serial=no \
alpine_repo=${alpine_mirror}/${alpine_branch}/main ssh_key=http://${url}/state/root-key.pub\"
-nic tap,model=virtio-net-pci,ifname=${tap},script=no,downscript=no,mac=${mac} 
${9pargs}  -display none"
`
)

type vmWorker struct {
	cueball.Executor
	Name string
	Port int
	RePull bool
	Wan  net.Interface
	Bridge  net.Interface
	Upstream net.Addr
	// break out/generalize
	AlpineMirror *url.URL
	AlpineBranch string
	AlpineArch string // narrow?
	client *http.Client
}

func (s *vmWorker) Name() string {
	return "vm-worker"
}

func NewVmWorker() *vmWorker { // concrete type permits field setting
	w := new(vmWorker)
	w.Executor = worker.NewExecutor(retry.NewCount(3, w.pullBinaries)...)
	w.client = new(http.Client)
	return w
}

func (w *vmWorker) pullBinaries(ctx context.Context, o cueball.ObjectStore) error {
	for _, u := range binaries {
		if ! w.Exists(w.key(u)) || w.RePull {
			resp, err := http.Get(w.AlpineMirror.JoinPath(w.AlpineBranch,
				"releases", w.AlpineArch, "netboot", u).String())
			if err != nil {
				return err
			}
			o.Save(w.key(u), resp.Body)
		}
	}
	return nil
}

func (w *vmWorker) key(suf string) string {
	x := []string{w.Name(), w.AlpineBranch, w.AlpineArch, suf}
	return strings.Join(x, sep)
}

func (s *vmWorker) bridge(ctx context.Context) error {
	return nil
}

Vagrant.configure("2") do |config|
  config.vm.define "docker-1" do |host|
    host.vm.hostname = 'docker-1'
    host.vm.box = "ubuntu/xenial64"

    host.vm.provider "virtualbox" do |v|
      v.memory = 512
      v.cpus = 2
    end

    host.vm.synced_folder "./", "/opt/sand"
    host.vm.network "private_network", ip: "192.168.254.2"
  end

  config.vm.define "docker-2" do |host|
    host.vm.hostname = 'docker-2'
    host.vm.box = "ubuntu/xenial64"

    host.vm.provider "virtualbox" do |v|
      v.memory = 1024
      v.cpus = 2
    end

    host.vm.synced_folder "./", "/opt/sand"
    host.vm.network "private_network", ip: "192.168.254.3"
  end

  config.vm.provision "docker"
end

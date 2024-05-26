postcreate: envconfig editor

envconfig:
	cd ~
	git clone --recursive --depth=1 https://github.com/nnishant776/dotfiles ~/dotfiles
	cp ~/dotfiles/.profile ~/.profile
	cp ~/dotfiles/.bashrc ~/.bashrc
	rm -rf ~/dotfiles

editorconfig:
	cd ~ 
	rm -rf ~/.config/nvim
	git clone --recursive --depth=1 -b dev https://github.com/nnishant776/init.lua ~/.config/nvim

editortools:
	cd ~
	curl -G -L https://github.com/BurntSushi/ripgrep/releases/download/13.0.0/ripgrep_13.0.0_amd64.deb -o ~/ripgrep_13.0.0_amd64.deb
	curl -G -L https://github.com/sharkdp/fd/releases/download/v8.7.0/fd-musl_8.7.0_amd64.deb -o ~/fd-musl_8.7.0_amd64.deb
	apt -y install ~/ripgrep_13.0.0_amd64.deb
	apt -y install ~/fd-musl_8.7.0_amd64.deb
	rm -rf ~/fd-musl_8.7.0_amd64.deb
	rm -rf ~/ripgrep_13.0.0_amd64.deb
	go install "github.com/cweill/gotests/gotests@latest"
	go install "github.com/fatih/gomodifytags@latest"
	go install "github.com/josharian/impl@latest"
	go install "honnef.co/go/tools/cmd/staticcheck@latest"
	go install "golang.org/x/tools/gopls@latest"
	go install "github.com/segmentio/golines@latest"
	go install "golang.org/x/tools/cmd/goimports@latest"

editorapp:
	curl -G -L https://github.com/neovim/neovim/releases/download/stable/nvim-linux64.tar.gz -o ~/nvim-linux64.tar.gz
	mkdir -p ~/nvim-linux64
	tar -C ~/nvim-linux64 --strip-components=1 -xf ~/nvim-linux64.tar.gz
	rm -rf ~/nvim-linux64/man
	cp -rf ~/nvim-linux64/* /usr/local/
	rm -rf ~/nvim-linux64.tar.gz
	rm -rf ~/nvim-linux64

editor: editorconfig editortools editorapp
	env features=ide nvim --headless "+Lazy restore" "+TSUpdateSync" "+qa"

#!/bin/bash

GREEN='\e[32m'
RED='\e[31m'
RESET='\e[0m'

install_path="/bin/syno_cpuinfo"
download_url="https://gh-proxy.com/https://github.com/GroverLau/syno_cpuinfo/releases/latest/download/syno_cpuinfo"
#https://github.com/GroverLau/syno_cpuinfo/releases/latest/download/syno_cpuinfo

print() {
    local color="$1"
    local text="$2"

    case "$color" in
        "r")
            echo -e "${RED}$text${RESET}"
            ;;
        "g")
            echo -e "${GREEN}$text${RESET}"
            ;;
        *)
            echo $@
            ;;
    esac
}

download() {
    if [ -f "$install_path" ]; then
        print "检测到已安装"
        read -p "是否需要重新安装(y/N): " reinstall
        if [[ "$reinstall" =~ ^[Yy]$ ]]; then
            uninstall
        else
            print "脚本退出"
            exit 0
        fi
    fi
    print "下载主程序"
    wget -q --show-progress -O $install_path "$download_url"
    if [ $? -ne 0 ]; then
        print r "下载失败"
        exit 1
    fi
    chmod 0755 $install_path
    if [ $? -ne 0 ]; then
        print r "赋予执行权限失败：无法设置 $install_path 的权限."
        exit 1
    fi
}

replace(){
    print "备份nginx配置文件"
    cp -f /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak
    cp -f /usr/syno/share/nginx/nginx.mustache /usr/syno/share/nginx/nginx.mustache.bak
    print "修改nginx配置文件"
    sed -i 's|/run/synoscgi.sock;|/run/scgi_proxy.sock;|' /etc/nginx/nginx.conf
    sed -i 's|/run/synoscgi.sock;|/run/scgi_proxy.sock;|' /usr/syno/share/nginx/nginx.mustache
    print "重载nginx配置文件"
    systemctl reload nginx
    if systemctl status nginx &>/dev/null; then
        print g "nginx运行中.."
        print g "脚本执行完成"
    else
        print r "nginx未运行,恢复配置."
        cp -f /etc/nginx/nginx.conf.bak /etc/nginx/nginx.conf
        cp -f /usr/syno/share/nginx/nginx.mustache.bak /usr/syno/share/nginx/nginx.mustache
        systemctl reload nginx
        print "执行失败,脚本退出."
        systemctl stop syno_cpuinfo &>/dev/null
        systemctl disable syno_cpuinfo &>/dev/null
        rm -r /lib/systemd/system/syno_cpuinfo.service  &>/dev/null
        systemctl daemon-reload 
        rm -r $install_path &>/dev/null
        rm -r /etc/syno_cpuinfo/config.conf &>/dev/null
        exit 1
    fi
}

install(){
    cat <<EOF > /lib/systemd/system/syno_cpuinfo.service 
[Unit]
Description=SCGI Proxy
After=network.target

[Service]
Type=simple
ExecStart=${install_path}
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=3
StartLimitInterval=60
StartLimitBurst=3

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable syno_cpuinfo
    systemctl start syno_cpuinfo
    if systemctl status syno_cpuinfo &>/dev/null; then
        print g "代理工具运行中.."
        replace
    else
        print r "代理工具未运行,脚本退出!"
        exit 1
    fi
}

input(){
    read -p "请输入 Vendor(eg. Intel/AMD): " vendor
    read -p "请输入 Family(eg. Core/Celeron): " family
    read -p "请输入 Series(eg. I5-8600T/J3455): " series
    read -p "请输入 Cores(eg. 6 / 6 + 6): " cores
    read -p "请输入 Speed(eg. 2300): " speed
    echo -e "\nVendor: ${GREEN}$vendor${RESET}"
    echo -e "Family: ${GREEN}$family${RESET}"
    echo -e "Series: ${GREEN}$series${RESET}"
    echo -e "Cores: ${GREEN}$cores${RESET}"
    echo -e "Speed: ${GREEN}$speed${RESET}\n"
    mkdir /etc/syno_cpuinfo/ 2>/dev/null
    cat <<EOF > /etc/syno_cpuinfo/config.conf 
Vendor =  $vendor
Family = $family
Series = $series
Cores = $cores
ClockSpeed = $speed
EOF
}

customize() {
    local need_customize

    if [ "$1" -ne 0 ]; then
        print r "获取 CPU 信息失败。"
        print "自定义 CPU 信息👇"
        need_customize="Y"
    else
        read -p "是否需要自定义 CPU 信息? (y/N): " need_customize
        need_customize=${need_customize:-N}
    fi

    if [[ "$need_customize" =~ ^[Yy]$ ]]; then
        input
    fi
}

uninstall(){
    print "准备卸载"
    sed -i 's|/run/scgi_proxy.sock;|/run/synoscgi.sock;|' /etc/nginx/nginx.conf
    sed -i 's|/run/scgi_proxy.sock;|/run/synoscgi.sock;|' /usr/syno/share/nginx/nginx.mustache
    systemctl reload nginx
    systemctl stop syno_cpuinfo &>/dev/null
    systemctl disable syno_cpuinfo &>/dev/null
    rm -r /lib/systemd/system/syno_cpuinfo.service  &>/dev/null
    systemctl daemon-reload 
    rm -r $install_path &>/dev/null
    rm -r /etc/syno_cpuinfo/config.conf &>/dev/null
    print "卸载完成"
}

checkRoot(){
    if [[ $( whoami ) != "root" ]]; then
        print r "请使用root权限运行脚本!"
        exit 1
    fi
}

checkInfo(){
    print "查看CPU温度"
    $install_path -t
    if [ "$?" -ne 0 ]; then
        print r "获取 CPU 温度信息失败,检查模块、驱动是否正确加载? 或者你可以尝试添加/etc/sensors3.conf配置文件"
        exit 1
    fi
    print "查看CPU信息"
    $install_path -i
    customize $?
}

reload(){
    print "更新中...."
    if systemctl reload syno_cpuinfo &>/dev/null; then
        print g "Ok,代理工具运行中.."
    else
        print r "更新CPU信息失败"
    fi
}

main() {
    case "$1" in
        "uninstall")
            print g "卸载"
            uninstall
            ;;
        "edit")
            print "编辑自定义CPU信息"
            input
            reload
            ;;
        *)
            print g "安装"
            download
            clear
            checkInfo
            install
            ;;
    esac

}
logo(){
    cat << "EOF"
----------------------------------------------------------------------------------------------
#     ____ __  __   _  __  ____        _____   ___   __  __        ____   _  __   ____  ____ 
#    / __/ \ \/ /  / |/ / / __ \      / ___/  / _ \ / / / /       /  _/  / |/ /  / __/ / __ \
#   _\ \    \  /  /    / / /_/ /     / /__   / ___// /_/ /       _/ /   /    /  / _/  / /_/ /
#  /___/    /_/  /_/|_/  \____/      \___/  /_/    \____/       /___/  /_/|_/  /_/    \____/ 
#                                                                                       @Lan's
----------------------------------------------------------------------------------------------
EOF
}
clear
logo
checkRoot
main $@

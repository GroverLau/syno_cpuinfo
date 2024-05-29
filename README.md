# 自用黑群 CPU 硬件信息及真实温度显示

## 简介

显示系统的真实 CPU 信息和温度。对后端进行代理并通过读取系统硬件信息，将 CPU 相关信息以及当前的系统温度嵌入到目标数据流中。支持自定义CPU型号.

## 功能

- 通过 /proc/cpuinfo 读取真实的 CPU 厂商、系列、核心数、时钟频率信息。
- 通过 /sys/class/thermal/thermal_zone0/temp 文件读取当前的CPU温度。
- 将处理后的数据写入到目标输出中。

## 实现原理

工具通过以下步骤实现功能：

1. **设置代理**: 监听 SCGI 套接字请求并更改nginx.conf后端服务器地址
1. **数据读取**：接收客户端请求转发至后端,读取数据并进行处理。
3. **数据替换**：替换数据中的特定字段，将实际的 CPU 信息和系统温度嵌入到数据中。
4. **数据写入**：将修改后的数据写入到客户端。

## 如何使用
1. 登录到SSH终端
2. 切换到root权限
   > sudo -i
3. 执行一键安装脚本
   > wget https://cdn.jsdelivr.net/gh/GroverLau/syno_cpuinfo/syno_cpuinfo.sh && bash syno_cpuinfo.sh

自定义CPU信号:
   > wget https://cdn.jsdelivr.net/gh/GroverLau/syno_cpuinfo/syno_cpuinfo.sh && bash syno_cpuinfo.sh edit

卸载:
   > wget https://cdn.jsdelivr.net/gh/GroverLau/syno_cpuinfo/syno_cpuinfo.sh && bash syno_cpuinfo.sh uninstall
   

## DSM系统显示
![DSM](img/1.jpg)

## 群晖助手
![群晖助手](img/2.jpg)

## 群晖管家
![群晖管家](img/3.jpg)
##  
![不同方式读取温度](img/4.jpg)

![自定义型号](img/5.jpg)
# uebuildtool
ue4游戏打包工具
LiteIDE打包go程序，去掉符号信息
   编译配置->自定义->BUILDFLAGS增加-ldflags "-s"
修改输出文件名称
   编译配置->自定义->BUILDFLAGS增加参数-o $(TARGETBASENAME)$(GOEXE), 之后修改TARGETBASENAME的值

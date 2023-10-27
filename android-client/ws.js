// 创建一个http client，可以设定client是否重连，心跳等功能
// 创建一个request 请求对象，采用什么协议ws 或wss 、服务器、端口都能内容
// 设置监听，当websocket 生命周期内的一些事情。
// 设置上面的操作以后，打开链接，创建webSocket 客户端。
// 用webSocket 客户端 发送消息 webSocket.send("你好服务器");

importPackage(Packages["okhttp3"]); //导入包

var client = new OkHttpClient.Builder().retryOnConnectionFailure(true).build();

//vscode  插件的ip地址
var request = new Request.Builder().url("ws://192.168.31.164:9317").build();

//清理一次
client.dispatcher().cancelAll();

myListener = {
  onOpen: function (webSocket, response) {
    print("onOpen");
    //打开链接后，想服务器端发送一条消息
    var json = {};
    json.type = "hello";
    json.data = {
      device_name: "模拟设备",
      client_version: 123,
      app_version: 123,
      app_version_code: "233",
    };
    var hello = JSON.stringify(json);
    webSocket.send(hello);
  },
  onMessage: function (webSocket, msg) {
    //msg可能是字符串，也可能是byte数组，取决于服务器送的内容
    print("msg");
    print(msg);
  },
  onClosing: function (webSocket, code, response) {
    print("正在关闭");
  },
  onClosed: function (webSocket, code, response) {
    print("已关闭");
  },
  onFailure: function (webSocket, t, response) {
    print("错误");
    print(t);
  },
};

var webSocket = client.newWebSocket(request, new WebSocketListener(myListener)); //创建链接

setInterval(() => {
  // 防止主线程退出
}, 1000);

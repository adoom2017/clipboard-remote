"ui";
importClass(android.graphics.Color);
importClass(android.graphics.drawable.GradientDrawable);
importClass(android.graphics.Rect);
ui.layout(
  <vertical id="sss" h="*">
    <frame bg="#404A56" h="*">
      <vertical h="*" align="center" margin="0 50" gravity="center">
        <card
          w="*"
          margin="10 5"
          h="150"
          cardCornerRadius="6dp"
          cardElevation="1dp"
          gravity="center_vertical"
          clickable="true"
        >
          <vertical>
            <text
              text="钉钉剪贴转发"
              gravity="center"
              textColor="black"
              textStyle="bold"
              size="16"
            />
            <text
              text="1.0.1"
              gravity="center"
              textColor="black"
              textStyle="bold"
              size="14"
            />
            <text textSize="13sp" textColor="black" text="                 " />
            <text textSize="13sp" textColor="black" text="                 " />
            <vertical w="*" clickable="true" marginLeft="9" marginRight="9">
              <button
                id="listener"
                w="*"
                layout_gravity="center"
                bg="#166BC6"
                text="开启监听"
                textSize="17dp"
                textColor="white"
              />
            </vertical>
          </vertical>
        </card>
      </vertical>
    </frame>
  </vertical>
);
ui.statusBarColor("#404A56");
let view;
view = ui.listener;
setBackgroundRoundGradientDottedRectanglebt(view);
function setBackgroundRoundGradientDottedRectanglebt(view) {
  gradientDrawable = new GradientDrawable();
  gradientDrawable.setShape(GradientDrawable.LINEAR_GRADIENT);
  gradientDrawable.setColor(colors.parseColor("#3369C1"));
  view.setBackground(gradientDrawable);
  gradientDrawable.setCornerRadius(14);
}

ui.listener.on("click", () => {
  plusOne();
  setInterval(() => {}, 1000);
});

function plusOne() {
  // 获取文本
  let text = ui.listener.text();
  log(text);
  // 解析为数字
  if (text == "开启监听") {
    // 设置文本
    toastLog("已经为您打开监听");
    setTimeout(function () {
      ui.listener.setText("关闭监听");
    }, 300);
    threads.start(clip_listener);
    //保持脚本运行
    setInterval(() => {}, 1000);
    // 1秒后继续
  } else {
    // 设置文本
    toastLog("已经为您关闭监听");
    setTimeout(function () {
      ui.listener.setText("开启监听");
    }, 300);
    threads.shutDownAll();
    // 1秒后继续
  }
}

importClass(android.content.ClipData.Item);
var clip_Timestamp = null;
function getClipstr() {
  var clipborad = context.getSystemService(context.CLIPBOARD_SERVICE);
  var clip = clipborad.getPrimaryClip();

  try {
    //时间戳
    if (clip_Timestamp != clip.getDescription().getTimestamp()) {
      if (clip_Timestamp == null) {
        //首次存储时间戳,下次进行比对
        clip_Timestamp = clip.getDescription().getTimestamp();
      } else {
        var item = clip.getItemAt(0);
        clip_Timestamp = clip.getDescription().getTimestamp();
        item = clip.getItemAt(0);
        log(item.getText()); //获取剪贴板内容
        //进行发送
        send_msg(item.getText());
      }
    }
  } catch (e) {}
}

function clip_listener() {
  importClass(android.os.Build);
  var version = Build.VERSION.RELEASE;
  log(version);
  if (version < 10) {
    //安卓10以下版本【推荐，不存在焦点抢占的情况】
    while (true) {
      sleep(100);
      getClipstr();
    }
  } else {
    //安卓10以上版本
    while (true) {
      sleep(4000);
      var w = floaty.window(<text />);
      ui.run(function () {
        w.requestFocus();
        setTimeout(() => {
          getClipstr();
          w.close();
        }, 1200);
      });
    }
  }
}


var url = "https://127.0.0.1/clipboard/get"
var username = "test"
var password = "password"

var authorization = "Basic " + $base64.encode(username+":"+password)

console.show();

var r = http.get(url, {
    headers: {
        'Authorization': authorization
    }
});
log("code = " + r.statusCode);
log("html = " + r.body.string());
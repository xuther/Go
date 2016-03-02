makePostCall = function  (url, data, callback){
  var json_data = JSON.stringify(data);
   $.ajax({
     type: "POST",
     url: url,
     data: json_data,
     contentType: "application/json;charset=utf-8"
   }).success(callback)
}

login = function(){
  username = $("#loginUsername").val();
  password = $("#loginPassword").val();

  console.log(username)
  console.log(password)

  loginCall(username,password)
}

getData = function(){

  checkLogin()
  getMessages()
  setTimeout(getData, 15000);
}

getMessages = function(){
  $.ajax({
    type: "GET",
    url: "api/getMessages",
    contentType: "application/json;charset=utf-8"
  }).always(function(data) {
    console.log("Messages")
    console.log(data)
    displayMessages(data)
  })
}

displayMessages = function(data){
  var obj = JSON.parse(data)
  console.log(obj)
  var table = document.getElementById("messageTable")
  table.getElementsByTagName("tbody")[0].innerHTML = table.rows[0].innerHTML;
  for (var i = 0; i < obj.length; i++){
    var row = table.insertRow(1);
    var cell1 = row.insertCell(0);
    var cell2 = row.insertCell(1);

    cell1.innerHTML = obj[i].Originator
    cell2.innerHTML = obj[i].Text
  }
  console.log(obj.length)
  document.getElementById("messageNo").innerHTML = obj.length
}

checkLogin = function(){
  $.ajax({
    type: "GET",
    url: "api/checkLogin",
    contentType: "application/json;charset=utf-8"
  }).always(function(data) {
    console.log("Messages")
    console.log(data)
    displayMessages(data)
  })
}

parseAllData = function(data){
  if(data == "false"){
    $(".data").css('visibility','hidden');
    $(".LogInPrompt").css('visibility','visible');
  }
  else {
    $(".data").css('visibility','visible');
    $(".LogInPrompt").css('visibility','hidden')
  }
}

loginCall = function(username, password)
{
  console.log("logging in")

  makePostCall("api/login",
  { 'Username': username,
    'Password': password
  }, function(data) {
     loggedIn(data);
   })
}

redirectToOath = function(data) {
  //CHANGE FROM LOCALHOST
  window.location.href= 'https://foursquare.com/oauth2/authenticate' +
    "?client_id=5ATHFEOTK5EU23DGQXCJ4XHYF1OWTBDIIV2CHXBAYQN0X5IO"+
    "&response_type=code"+
    '&redirect_uri=https://localhost/landing.html'
}

getToken = function() {
  console.log(window.location.search)

  $.ajax({
    type: "GET",
    url: "api/getOathToken" + window.location.search,
    contentType: "application/json;charset=utf-8"
  }).always(
  function(data) {
    console.log("We got the token!!");
    console.log(data);
    window.location.href = '/'
  })
}

loggedIn = function(data) {
  console.log(data)
  if (data == "Success"){
    window.location.href= '/message.html';
  }
  else {
    alert("Login failed, please try again.");
  }
}

sendMessage = function() {
  console.log("seding a message")

  content = $("#messageData").val()
  console.log(content)
  
  $.ajax({
    type: "POST",
    url: "/api/sendMessage",
    data: content,
    contentType: "plain/text;charset=utf-8"
  }).success(function() {
    alert("message sent!")
  })

}

register = function() {
  console.log("Registering")

  username = $("#registerUsername").val();
  password = $("#registerPass").val();
  confirmPass = $("#registerConfirmPass").val();

  console.log("Password: " + password)
  console.log("Confirm Pass: " + confirmPass)

  if (username == "" || password == "" || password != confirmPass)
  {
    alert('Please check the fields to ensure they have entries, and the passwords match')
  }
  else
  {
    makePostCall("api/register",
    { 'Username': username,
      'Password': password
    }, function(data) {
      console.log(data);
      if(data =="Success")
        loginCall(username, password);
      else if (data =="Username Exists")
        alert("Username already exists, please select other")
      else
        alert("Register Failed");
    }
    )
  }

}

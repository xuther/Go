makePostCall = function  (url, data, callback){
  var json_data = JSON.stringify(data);
   $.ajax({
     type: "POST",
     url: url,
     data: json_data,
     contentType: "application/json;charset=utf-8"
   }).success(callback);
}

login = function(){


  username = $("#loginUsername").val();
  password = $("#loginPassword").val();

  console.log(username)
  console.log(password)

  loginCall(username,password)
}

loginCall = function(username, password)
{
  console.log("logging in")

  makePostCall("api/login",
  { 'Username': username,
    'Password': password
  }, function(data) {
     console.log(data);
     loggedIn(data);
   })
}

loggedIn = function(data) {
  console.log(data)
    if (data == "Success"){
    window.location.href= '/';
  }
  else {
    alert("Login failed, please try again.");
  }
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

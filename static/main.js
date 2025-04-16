// Check if an URL exists
function checkUrl(url) {
	var http = new XMLHttpRequest();
	http.open('HEAD', url, false);
	http.send();
	return http.status == 200;
}



// Shortcut to get element
function element(element) {
	return document.getElementById(element);
}



// Display an notification notification
function displayError(message) {
	let notification = document.createElement('div');

	// Set the ID to always replace the same element with the notification
	notification.setAttribute("id", "error");

	notification.innerHTML = message;
	notification.className = 'notification notification-error';

	document.body.appendChild(notification);

	// Automatically delete notification after few seconds
	setTimeout(function () {
		document.body.removeChild(notification);
	}, 3000);

}




// Display an notification notification
function displayInfo(message) {
	let notification = document.createElement('div');

	// Set the ID to always replace the same element with the notification
	notification.setAttribute("id", "info");

	notification.innerHTML = message;
	notification.className = 'notification notification-info';

	document.body.appendChild(notification);

	// Automatically delete notification after few seconds
	setTimeout(function () {
		document.body.removeChild(notification);
	}, 3000);

}



// // Set logo from given file
// function setLogo(fileLogo) {
// 	let logoImg = document.createElement('object');
// 	logoImg.setAttribute('type', 'image/svg+xml');
// 	logoImg.setAttribute('data', fileLogo);

// 	let logo = document.getElementById('logo');
// 	logo.appendChild(logoImg)
// }
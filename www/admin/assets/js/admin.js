function print(pin, price) {
	if (confirm("Print Job with PIN "+pin+" for "+price+" \u20AC?")) {
		var data = new FormData();
		data.append('pin', pin);
		//data.append('internal', '1');

		var xhr = new XMLHttpRequest();
        	xhr.open('POST', "/print", true);
        	xhr.onload = function() {
            		if (this.status == 200) {
                		alert("Success: " + xhr.response);
            		} else {
				alert("FAIL: " + xhr.response);
            		}
        	};
        	xhr.send(data);
	}
}

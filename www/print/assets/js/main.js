/*!
Copyright (c) 2015 Julien Schmidt


Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without
restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the
Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
const haspaStatusURL = 'http://hackerspace.stusta.de/current.json';

(function() {
    let dragover = false;
    let selected = false;
    let uf;
    let file;
    let fileinfo;
    let filedrop;
    let submit;

    // document.getElementById shortcut
    function $(id) {
        return document.getElementById(id);
    }

    function radioValue(name) {
        const radios = document.getElementsByName(name);
        for (let i = 0, length = radios.length; i < length; i++) {
            if (radios[i].checked) {
                return radios[i].value;
            }
        }
    }

    function fetchOpeningHours() {
        // not working because of no cors
        const xhr =  new XMLHttpRequest();
        xhr.open('GET', haspaStatusURL, true);
        xhr.onload = function() {
            if (this.status === 200) {
                alert("Haspa status " + xhr.response);
            } else {
                alert("Haspa fail " + xhr.response);
            }
        };
        xhr.send()
    }

    function over(e) {
        e.stopPropagation();
        e.preventDefault();
        if (!dragover) {
            dragover = true;
            filedrop.className = "hover";
        }
    }

    function leave(e) {
        e.preventDefault();
        filedrop.className = (selected ? "selected" : "");
        dragover = false;
    }

    function select(e) {
        leave(e);

        // fetch FileList object
        const files = e.target.files || e.dataTransfer.files;
        uf = files[0];
        let type = uf.type;
        if (type === "application/x-pdf" || type === "text/pdf") {
            type = "application/pdf";
        } else if (type === "application/save-as" || type === "application/x-save-as") {
            if (uf.name.toLowerCase().substr(uf.name.lastIndexOf('.') + 1) === "pdf") {
                type = "application/pdf";
            }
        }

        if (type !== "application/pdf") {
            alert(errFileType);
            uf = undefined;
            return;
        }

        // Check file size
        if (uf.size >= $("MAX_FILE_SIZE").value) {
            alert(errFileSize);
            uf = undefined;
            return;
        }

        selected = true;
        document.body.className = "step2";

        fileinfo.innerHTML = "<strong>" + uf.name +
            "</strong></br />Size: <strong>" + uf.size +
            "</strong> Bytes";
        filedrop.className = "selected";
    }

    function send() {
        document.body.className = "progress";
        submit.disabled = false;
        let fd = new FormData();
        fd.append("file", uf, uf.name);
        if ($("bw").checked) {
            fd.append("bw", $("bw").value);
        }
        fd.append("duplex", radioValue("duplex"));
        fd.append("copies", $("copies").value);

        let xhr = new XMLHttpRequest();
        xhr.open('POST', upload.action, true);

        /*xhr.upload.onprogress = function(e) {
            if (e.lengthComputable) {
                var percentComplete = (e.loaded / e.total) * 100;
                console.log(percentComplete + '% uploaded');
            }
        };
        xhr.onreadystatechange = function () {
            $("result").innerHTML = xhr.response;
        };*/
        xhr.onload = function() {
            if (this.status === 200) {
                $("result-content").innerHTML = xhr.response;
                document.body.className = "result";
            } else {
                document.body.className = "error";
                $("progress").innerHTML = "<h3>" + xhr.response + "</h3>";
            }
        };
        xhr.send(fd);
    }


    // initialize
    function init() {
        file = $("file");
        fileinfo = $("fileinfo");
        filedrop = $("filedrop");
        submit = $("submit");
        upload = $("upload");

        file.addEventListener("change", select, false);
        upload.addEventListener("submit", function(ev) {
            ev.preventDefault();
            submit.disabled = true;
            send();
            return false;
        });
        filedrop.onclick = function() {
            file.click();
        };

        if ((new XMLHttpRequest()).upload) {
            // file drop
            filedrop.addEventListener("dragover", over, false);
            filedrop.addEventListener("dragleave", leave, false);
            filedrop.addEventListener("drop", select, false);
            document.body.className = "step1";
        }

    }

    if (window.File && window.FileList && window.FileReader) {
        init();
    }
})();


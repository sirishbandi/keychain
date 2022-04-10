console.log("converter running")
document.getElementById('inp').onchange = function (e) {
    var img = new Image();
    img.onload = uploadImage;
    img.onerror = failed;
    img.src = URL.createObjectURL(this.files[0]);
};

function uploadImage() {

    // Initialize some variables

    var forceDither = true;

    var canvas = document.querySelector('canvas');
    var context = canvas.getContext('2d');
    var width = 200 * (this.width / this.height) - (200 * (this.width / this.height) % 4);
    var height = 200;

    canvas.width = width;
    canvas.height = height;

    // Draw our image, so we can read colors from it
    context.drawImage(this, 0, 0, width, height);
    context.imageSmoothingQuality = 'high';
    mono = canvas2based64monobitmap(canvas);
    console.log("Final img", mono);
    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/keychain/post');
    xhr.send(mono);

    alert("Image Added.")
}

function failed() {
    alert("The provided file couldn't be loaded as an Image media");
}
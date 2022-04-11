#! /bin/bash

echo "Coverting Image"
convert channel.png -dither FloydSteinberg -define dither:diffusion-amount=85% -monochrome -colors 2 channel.bmp

echo "Uploading Image"
gsutil -o "GSUtil:parallel_process_count=1" cp channel.bmp gs://keychainbucket/
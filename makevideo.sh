#!/bin/zsh

ffmpeg -r 50 -i ./frames/FRAME_%d.png -vf "pad=ceil(iw/2)*2:ceil(ih/2)*2" -vcodec libx264 -crf 25 -pix_fmt yuv420p out.mp4

ffmpeg  -i out.mp4 -i ./resources/test5.wav -c:v copy -map 0:v -map 1:a -y out-audio.mp4

#!/bin/bash

rm -rf /etc/slurm
cp -r -f  /tmp/slurm/  /etc

nvidia_device=$(ls /dev | grep -E '^nvidia[0-9]+$')
echo $nvidia_device

# 检查变量是否为空
if [ -n "$nvidia_device" ]; then
  # 创建或覆盖 gres.conf 文件
  cat <<EOF > /etc/slurm/gres.conf
Name=gpu Type=nvidia File=/dev/${nvidia_device}
Name=shard Count=8 File=/dev/${nvidia_device}
EOF
fi

/usr/sbin/slurmd -D

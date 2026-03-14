echo "=== 即将删除的空目录 ===" 
find . -type d -empty ! -path "*/.git*" ! -path "*/node_modules*" ! -path "*/.next*" | head -20
echo "..."
find . -type d -empty ! -path "*/.git*" ! -path "*/node_modules*" ! -path "*/.next*" | wc -l

read -p "确认删除以上目录？(y/n) " confirm
if [ "$confirm" = "y" ]; then
    find . -type d -empty ! -path "*/.git*" ! -path "*/node_modules*" ! -path "*/.next*" -exec rmdir {} +
    echo "删除完成"
fi

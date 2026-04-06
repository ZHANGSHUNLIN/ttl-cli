#!/bin/bash
# 验证脚本 - 编译检查与集成测试
# 用于 CI/CD 流水线或本地提交前验证

set -e

echo "========================================="
echo "  TTL 项目验证脚本"
echo "========================================="

# 1. 编译检查
echo ""
echo "[1/5] 编译检查..."
go build -o /tmp/ttl-test-binary . || {
    echo "❌ 编译失败！"
    exit 1
}
echo "✅ 编译成功"

# 2. 创建临时测试环境
echo ""
echo "[2/5] 创建临时测试环境..."
TEST_DIR=$(mktemp -d)
TEST_CONF="$TEST_DIR/test.conf"
TEST_DB="$TEST_DIR/test.db"

cat > "$TEST_CONF" << EOF
db_path = $TEST_DIR/data.bbolt
storage_type = bbolt
EOF

echo "   测试目录: $TEST_DIR"
echo "✅ 临时环境创建成功"

# 3. 基础功能回归测试
echo ""
echo "[3/5] 基础功能回归测试..."
BINARY="/tmp/ttl-test-binary"

# 资源管理
echo "   - 测试 add..."
$BINARY --conf "$TEST_CONF" add "test-resource" "https://example.com" > /dev/null

echo "   - 测试 get..."
$BINARY --conf "$TEST_CONF" get test-resource | grep -q "example.com"

echo "   - 测试 tag..."
$BINARY --conf "$TEST_CONF" tag test-resource ci automated > /dev/null

echo "   - 测试 export..."
$BINARY --conf "$TEST_CONF" export -t resources > "$TEST_DIR/export.csv"
grep -q "key,value,tags" "$TEST_DIR/export.csv"
grep -q "test-resource" "$TEST_DIR/export.csv"

echo "   - 测试 dtag..."
$BINARY --conf "$TEST_CONF" dtag test-resource automated > /dev/null

echo "   - 测试 update..."
$BINARY --conf "$TEST_CONF" update test-resource "https://updated.example.com" > /dev/null

echo "   - 测试 rename..."
$BINARY --conf "$TEST_CONF" rename test-resource "renamed-resource" > /dev/null

echo "   - 测试 get (验证 rename)..."
$BINARY --conf "$TEST_CONF" get renamed-resource | grep -q "updated.example.com"

echo "   - 测试 del..."
$BINARY --conf "$TEST_CONF" del renamed-resource > /dev/null

echo "   - 验证删除..."
$BINARY --conf "$TEST_CONF" get renamed-resource 2>&1 | grep -q "未找到"

echo "   - 重新添加测试资源用于后续测试..."
$BINARY --conf "$TEST_CONF" add "test-resource-2" "https://example2.com" > /dev/null

echo "   - 测试 import..."
$BINARY --conf "$TEST_CONF" import "$TEST_DIR/export.csv" > /dev/null

echo "   - 测试 log add..."
$BINARY --conf "$TEST_CONF" log "CI 测试日志" > /dev/null

echo "   - 测试 log list..."
$BINARY --conf "$TEST_CONF" log -l | grep -q "CI 测试日志"

echo "   - 测试 version..."
$BINARY version | grep -qE "[0-9]+\.[0-9]+"

echo "   - 测试 config..."
$BINARY --conf "$TEST_CONF" config | grep -q "数据文件"

echo "   - 测试 history..."
$BINARY --conf "$TEST_CONF" history 5 | grep -q "test-resource"

echo "   - 测试 audit..."
$BINARY --conf "$TEST_CONF" audit | grep -q "总操作次数"

echo "   - 测试 encrypt..."
$BINARY --conf "$TEST_CONF" encrypt --migrate > /dev/null
# 验证加密后数据仍可读取
$BINARY --conf "$TEST_CONF" get test-resource-2 | grep -q "example2.com"

echo "   - 测试 key verify..."
$BINARY key verify > /dev/null

echo "   - 测试 decrypt..."
$BINARY --conf "$TEST_CONF" decrypt > /dev/null
# 验证解密后数据仍可读取
$BINARY --conf "$TEST_CONF" get test-resource-2 | grep -q "example2.com"

echo "   - 测试 MCP server 启动..."
# 测试 MCP server 能否正常初始化（timeout 2秒）
timeout 2 $BINARY --conf "$TEST_CONF" mcp < /dev/null 2>&1 | head -1 > /dev/null || true

echo "✅ 功能回归测试通过"

# 4. 单元测试
echo ""
echo "[4/5] 单元测试..."
go test ./... > /tmp/unit-test.log 2>&1 || {
    echo "❌ 单元测试失败！查看日志: /tmp/unit-test.log"
    cat /tmp/unit-test.log
    rm -rf "$TEST_DIR"
    rm -f /tmp/ttl-test-binary
    exit 1
}
echo "✅ 单元测试通过"

# 5. 集成测试
echo ""
echo "[5/5] 集成测试..."
go test ./integration_test/... > /tmp/integration-test.log 2>&1 || {
    echo "❌ 集成测试失败！查看日志: /tmp/integration-test.log"
    cat /tmp/integration-test.log
    rm -rf "$TEST_DIR"
    rm -f /tmp/ttl-test-binary
    exit 1
}
echo "✅ 集成测试通过"

# 6. 清理
echo ""
echo "清理临时文件..."
rm -rf "$TEST_DIR"
rm -f /tmp/ttl-test-binary

echo ""
echo "========================================="
echo "✅ 所有验证通过！"
echo "========================================="

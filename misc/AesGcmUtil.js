const crypto = require('crypto');

const ALGO = 'aes-256-gcm';
const IV_LEN = 12;

// key: Buffer,长度32字节
function encrypt(plainText, key) {
  const iv = crypto.randomBytes(IV_LEN);
  const cipher = crypto.createCipheriv(ALGO, key, iv);
  const encrypted = Buffer.concat([cipher.update(plainText, 'utf8'), cipher.final()]);
  const tag = cipher.getAuthTag(); // 16字节
  // 输出:iv + 密文 + tag
  return Buffer.concat([iv, encrypted, tag]).toString('base64');
}

function decrypt(base64Data, key) {
  const data = Buffer.from(base64Data, 'base64');
  const iv = data.subarray(0, IV_LEN);
  const tag = data.subarray(data.length - 16);
  const encrypted = data.subarray(IV_LEN, data.length - 16);

  const decipher = crypto.createDecipheriv(ALGO, key, iv);
  decipher.setAuthTag(tag);
  const decrypted = Buffer.concat([decipher.update(encrypted), decipher.final()]);
  return decrypted.toString('utf8');
}

// ---- 测试 ----
const key = Buffer.from('d8eab717abeca26cf5d0af2e216fa9f4'.slice(0, 32)); // 示例,实际请随机生成
const cipherText = encrypt('hello world', key);
console.log('加密结果:', cipherText);
console.log('解密结果:', decrypt(cipherText, key));
import javax.crypto.Cipher;
import javax.crypto.spec.GCMParameterSpec;
import javax.crypto.spec.SecretKeySpec;
import java.security.SecureRandom;
import java.util.Base64;
import java.nio.charset.StandardCharsets;
import java.nio.ByteBuffer;

public class AesGcmUtil {

    private static final String ALGO = "AES/GCM/NoPadding";
    private static final int IV_LEN = 12;
    private static final int TAG_LEN_BIT = 128; // 16字节 * 8

    public static String encrypt(String plainText, byte[] key) throws Exception {
        byte[] iv = new byte[IV_LEN];
        new SecureRandom().nextBytes(iv);

        Cipher cipher = Cipher.getInstance(ALGO);
        SecretKeySpec keySpec = new SecretKeySpec(key, "AES");
        GCMParameterSpec spec = new GCMParameterSpec(TAG_LEN_BIT, iv);
        cipher.init(Cipher.ENCRYPT_MODE, keySpec, spec);

        byte[] encrypted = cipher.doFinal(plainText.getBytes(StandardCharsets.UTF_8));
        // encrypted 已经包含 tag(Java 的实现会自动把tag拼在密文后面)

        ByteBuffer buffer = ByteBuffer.allocate(iv.length + encrypted.length);
        buffer.put(iv).put(encrypted);
        return Base64.getEncoder().encodeToString(buffer.array());
    }

    public static String decrypt(String base64Data, byte[] key) throws Exception {
        byte[] data = Base64.getDecoder().decode(base64Data);
        byte[] iv = new byte[IV_LEN];
        System.arraycopy(data, 0, iv, 0, IV_LEN);

        byte[] encrypted = new byte[data.length - IV_LEN];
        System.arraycopy(data, IV_LEN, encrypted, 0, encrypted.length);

        Cipher cipher = Cipher.getInstance(ALGO);
        SecretKeySpec keySpec = new SecretKeySpec(key, "AES");
        GCMParameterSpec spec = new GCMParameterSpec(TAG_LEN_BIT, iv);
        cipher.init(Cipher.DECRYPT_MODE, keySpec, spec);

        byte[] decrypted = cipher.doFinal(encrypted);
        return new String(decrypted, StandardCharsets.UTF_8);
    }

    public static void main(String[] args) throws Exception {
        byte[] key = "d8eab717abeca26cf5d0af2e216fa9f4".substring(0, 32).getBytes(StandardCharsets.UTF_8);
        String cipherText = encrypt("hello world", key);
        System.out.println("加密结果: " + cipherText);
        System.out.println("解密结果: " + decrypt(cipherText, key));
    }
}
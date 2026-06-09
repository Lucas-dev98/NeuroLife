import subprocess

flutter = r"C:\Users\lucas.bastos\Downloads\flutter_windows_3.44.1-stable\flutter\bin\flutter.bat"
jdk = r"C:\Users\lucas.bastos\Downloads\Java\jdk-17"
subprocess.run([flutter, "config", f"--jdk-dir={jdk}"], check=False)

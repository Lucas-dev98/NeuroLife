import subprocess

flutter = r"C:\Users\lucas.bastos\Downloads\flutter_windows_3.44.1-stable\flutter\bin\flutter.bat"
subprocess.run([flutter, "doctor", "-v"], check=False)

from flask import Flask, flash, request, redirect, url_for, send_from_directory, Response
from flask import render_template
import os
import zipfile, tarfile
import glob
from werkzeug.utils import secure_filename
import shlex, subprocess
import requests
from requests.structures import CaseInsensitiveDict
import signal, sys


def signal_handler(sig, frame):
    print("Exiting gracefully")
    cmds = shlex.split("./kill_urbit.sh")
    print(cmds)
    p = subprocess.Popen(cmds,shell=True)
    sys.exit(0)


urbit_url = "http://127.0.0.1:12321"
urbit_headers = CaseInsensitiveDict()
urbit_headers["Content-Type"] = "application/json"

urbit_code_data = '{ "source": { "dojo": "+code" }, "sink": { "stdout": null } }'
urbit_resetcode_data = '{ "source": { "dojo": "+hood/code %reset" }, "sink": { "app": "hood" } }'


UPLOAD_KEY = '/data/keys'
UPLOAD_PIER = '/data/piers'
AMES_PORT = 34343

app = Flask(__name__)
app.config['UPLOAD_KEY'] = UPLOAD_KEY
app.config['UPLOAD_PIER'] = UPLOAD_PIER
app.config['AMES_PORT'] = AMES_PORT

timeout = None
@app.route("/")
def hello():
    global timeout

    code = get_code()
    
    used_timeout = timeout
    timeout = None

    if code == None:
        return render_template('hello.html', piers=get_piers(), keys=get_keys(), code=code, timeout=used_timeout)
    else:
        return render_template('urbit_control.html', code=code, timeout=used_timeout)

@app.route('/stop_urbit', methods=['GET','POST'])
def stop_urbit():
    if request.method == 'POST':
        cmds = shlex.split("./kill_urbit.sh")
        print(cmds)
        p = subprocess.Popen(cmds,shell=True)
        timeout = 10000
    
    return redirect("/")

@app.route('/reset_code', methods=['GET','POST'])
def reset_code():
    if request.method == 'POST':
        try:
            resp = requests.post(urbit_url, headers=urbit_headers, data=urbit_resetcode_data)
        except requests.ConnectionError:
            pass
    return redirect("/")


def get_keys():
    keys = glob.glob(os.path.join(app.config['UPLOAD_KEY'], '*.key'))
    return keys

def get_piers():
    piers = glob.glob(os.path.join(app.config['UPLOAD_PIER'], '*/'))
    return piers

def get_code():
    try:
        resp = requests.post(urbit_url, headers=urbit_headers, data=urbit_code_data)
        return resp.json()
    except requests.ConnectionError:
        return None

@app.route('/boot', methods=['GET','POST'])
def boot():
    if request.method == 'POST':
        pier = request.form['boot']
        if pier.endswith('key'):
            # Boot up a new pier with keyfile
            cmd = './boot_key.sh %s %s'%(pier, AMES_PORT)
            timeout = 60*5*1000
            
            pass
        elif pier.endswith('/'):
            # Boot up the old pier
            cmd = './boot_pier.sh %s %s'%(pier, AMES_PORT)
            timeout = 10000
            pass
        cmds = shlex.split(cmd)
        p = subprocess.Popen(cmds)
    return redirect("/")


@app.route('/boot_new_comet', methods=['GET', 'POST'])
def boot_new_comet():
    cmd = './boot_new_comet.sh %s'%(AMES_PORT)
    print(cmd)
    cmds = shlex.split(cmd)
    p = subprocess.Popen(cmds)
    timeout = 20*5*1000
    return redirect("/")

@app.route('/upload_key', methods=['GET', 'POST'])
def upload_key():
    if request.method == 'POST':
        # check if the post request has the file part
        if 'file' not in request.files:
            flash('No file part')
            return redirect(request.url)
        file = request.files['file']
        # If the user does not select a file, the browser submits an
        # empty file without a filename.
        if file.filename == '':
            flash('No selected file')
            return redirect(request.url)
        filename = secure_filename(file.filename)
        file.save(os.path.join(app.config['UPLOAD_KEY'], filename))
    return redirect("/")


@app.route('/upload_pier', methods=['GET', 'POST'])
def upload_pier():
    if request.method == 'POST':
        # check if the post request has the file part
        if 'file' not in request.files:
            flash('No file part')
            return redirect(request.url)
        file = request.files['file']
        # If the user does not select a file, the browser submits an
        # empty file without a filename.
        if file.filename == '':
            flash('No selected file')
            return redirect(request.url)
        filename = secure_filename(file.filename)
        fn = os.path.join(app.config['UPLOAD_PIER'],filename)
        file.save(fn)

        if filename.endswith("zip"):
            with zipfile.ZipFile(fn) as zip_ref:
                zip_ref.extractall(app.config['UPLOAD_PIER']);
        elif filename.endswith("tar.gz") or filename.endswith("tgz"):
            tar = tarfile.open(fn,"r:gz")
            tar.extractall(app.config['UPLOAD_PIER'])
            tar.close()


        os.remove(os.path.join(app.config['UPLOAD_PIER'], filename))
    return redirect("/")



if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0')

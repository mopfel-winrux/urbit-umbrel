from flask import Flask, flash, request, redirect, url_for, send_from_directory
from flask import render_template
import os
import zipfile, tarfile
import glob
from werkzeug.utils import secure_filename
import shlex, subprocess
import requests
from requests.structures import CaseInsensitiveDict

urbit_url = "http://127.0.0.1:12321"
urbit_headers = CaseInsensitiveDict()
urbit_headers["Content-Type"] = "application/json"

urbit_code_data = '{ "source": { "dojo": "+code" }, "sink": { "stdout": null } }'


UPLOAD_KEY = './keys'
UPLOAD_PIER = './piers'
AMES_PORT = 34343

app = Flask(__name__)
app.config['UPLOAD_KEY'] = UPLOAD_KEY
app.config['UPLOAD_PIER'] = UPLOAD_PIER
app.config['AMES_PORT'] = AMES_PORT

@app.route("/")
def hello_world():
    return render_template('hello.html', piers=get_piers(), keys=get_keys(), code=get_code())


def get_keys():
    keys = glob.glob(os.path.join(app.config['UPLOAD_KEY'], '*.key'))
    return keys

def get_piers():
    piers = glob.glob(os.path.join(app.config['UPLOAD_PIER'], '*/'))
    return piers

def get_code():
    try:
        resp = requests.post(urbit_url, headers=urbit_headers, data=urbit_code_data)
        print(resp)
        return resp.json()
    except requests.ConnectionError:
        return None

@app.route('/boot', methods=['GET','POST'])
def boot():
    if request.method == 'POST':
        pier = request.form['boot']
        if pier.endswith('key'):
            #TODO Boot up a new pier with keyfile
            cmd = './boot_key.sh %s %s'%(pier, AMES_PORT)
            print(cmd)
            
            pass
        elif pier.endswith('/'):
            #TODO Boot up the old pier
            cmd = './boot_pier.sh %s %s'%(pier, AMES_PORT)
            print(cmd)
            pass
        cmds = shlex.split(cmd)
        p = subprocess.Popen(cmds,start_new_session=True)
    return redirect("/")


@app.route('/boot_new_comet', methods=['GET', 'POST'])
def boot_new_comet():
    cmd = './boot_new_comet.sh %s'%(AMES_PORT)
    print(cmd)
    # TODO: Write code that calls urbit comet bootup
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

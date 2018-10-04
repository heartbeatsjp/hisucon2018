import datetime
import shutil
from functools import wraps
import os
import random
import string
import markdown
from flask import request, redirect, url_for, render_template, \
    flash, Markup, jsonify, abort
from flask_login import login_user, logout_user, login_required, current_user
from werkzeug.security import generate_password_hash

from app import app, db
from app.form import LoginForm
from app.models import User, EduDaily, EduDailyComment


def admin_required(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        if not current_user.get_admin():
            flash('権限がありません。', 'danger')
            return redirect(url_for('index'))
        return f(*args, **kwargs)
    return decorated_function


@app.route('/reset', methods=['GET'])
def reset():
    db.session.query(EduDailyComment).delete()
    db.session.query(EduDaily).delete()
    db.session.query(User).delete()
    db.session.commit()

    db.session.execute(
        "LOAD DATA LOCAL INFILE "
        "'/home/hisucon/webapp/app/sql/hisucon201808-users.tsv' "
        "INTO TABLE users"
    )
    db.session.execute(
        "LOAD DATA LOCAL INFILE "
        "'/home/hisucon/webapp/app/sql/hisucon201808-edu_daily.tsv' "
        "INTO TABLE edu_daily"
    )
    db.session.execute(
        "LOAD DATA LOCAL INFILE "
        "'/home/hisucon/webapp/app/sql/hisucon201808-edu_daily_comment.tsv' "
        "INTO TABLE edu_daily_comment"
    )
    db.session.commit()

    edu_daily_path = os.path.join(app.root_path, 'edu_daily')
    static_path = os.path.join(app.root_path, 'static')

    if not os.path.isdir(edu_daily_path):
        os.mkdir(edu_daily_path)
    if not os.path.isdir(static_path):
        os.mkdir(static_path)

    edu_daily_org_path = os.path.join(app.root_path, 'edu_daily_org')
    static_org_path = os.path.join(app.root_path, 'static_org')

    for _from, _to in [(edu_daily_org_path, edu_daily_path),
                       (static_org_path, static_path)]:
        os.system("rsync -a --delete {}/ {}/".format(_from, _to))

    os.sync()

    return ('', 204)


@app.route('/login', methods=['GET', 'POST'])
def login():
    form = LoginForm(request.form)

    if request.method == 'POST':
        user = User.query.filter_by(name=form.name.data, is_deleted=0).first()
        if user is not None and user.check_password(form.password.data):
            login_user(user, remember=form.remember_me.data)
            flash('ログインに成功しました。', 'success')
            return redirect(url_for('index'))
        else:
            flash('ログインに失敗しました。', 'danger')
            return render_template('login.html', form=form), 403

    return render_template('login.html', form=form)


@app.route('/logout')
@login_required
def logout():
    logout_user()
    flash('ログアウトしました。', 'success')
    return redirect(url_for('login'))


@app.route('/')
def index():
    if current_user.is_authenticated:
        page = request.args.get('page', 1, type=int)
        edudailys = current_user.get_all_posts().paginate(page, 20, False)
        next_url = url_for(
            'index', page=edudailys.next_num) if edudailys.has_next else None
        prev_url = url_for(
            'index', page=edudailys.prev_num) if edudailys.has_prev else None

        return render_template('index.html', **locals())

    return redirect(url_for('login'))


@app.route('/edudaily/<int:uid>', methods=['GET'])
@login_required
def show_edu_daily(uid):
    edudaily = EduDaily.query.get(uid)

    if edudaily is None:
        abort(404)
    else:
        icon = User.query.filter_by(id=edudaily.user_id).first().icon
        textfile = os.path.join(app.root_path, 'edu_daily', edudaily.path)
        with open(textfile) as f:
            t = f.read()
        text = Markup(markdown.markdown(t, extensions=['gfm']))

        name_list = []
        iconlist = []
        commentlist = []
        edudailycomments = EduDailyComment.query.filter_by(
            edu_daily_id=edudaily.id).order_by(
                EduDailyComment.created_at.desc()).all()
        for edudailycomment in edudailycomments:
            commentfile = os.path.join(
                app.root_path, 'edu_daily', edudailycomment.path)
            with open(commentfile) as f:
                c = f.read()
            name_list.append(User.query.filter_by(
                id=edudailycomment.commenter_id).first().name)
            commentlist.append(
                Markup(markdown.markdown(c, extensions=['gfm'])))
            iconlist.append(User.query.filter_by(
                id=edudailycomment.commenter_id).first().icon)

        return render_template('text.html', **locals())


@app.route('/edudaily/new/', methods=['GET', 'POST'])
@login_required
def new():
    if request.method == 'POST':
        rand = ''.join(random.choices(
            string.ascii_letters + string.digits, k=20))
        filename = '{}_{}.md'.format(current_user.get_name(), rand)
        os_path = os.path.join(app.root_path, 'edu_daily', filename)
        ng_file_list = []

        if 'upload' in request.files:
            upload_files = request.files.getlist('upload')
            for upload_file in upload_files:
                if os.path.exists(os.path.join(app.root_path, 'static',
                                               upload_file.filename)):
                    ng_file_list.append(upload_file.filename)

            if ng_file_list:
                ng_file = ', '.join(ng_file_list)
                flash('{} はすでに同名のファイルが存在します。'.format(ng_file), 'danger')
                return redirect(request.url)
            else:
                for upload_file in upload_files:
                    upload_file.save(os.path.join(
                        app.root_path, 'static', upload_file.filename))

        with open(os_path, mode='w') as f:
            f.write(request.form['text'])

        edudaily = EduDaily(
            user_id=current_user.get_id(),
            title=request.form['title'],
            path=filename
        )

        db.session.add(edudaily)
        db.session.commit()

        flash('新規投稿が完了しました。', 'success')
        return redirect(url_for('index'))

    return render_template('new.html')


@app.route('/edudaily/<int:id>/new_com/', methods=['GET', 'POST'])
@login_required
def new_com(id):
    edu_daily_id = id
    if request.method == 'POST':
        rand = ''.join(random.choices(
            string.ascii_letters + string.digits, k=20))
        filename = '{}_{}.md'.format(current_user.get_name(), rand)
        os_path = os.path.join(app.root_path, 'edu_daily', filename)

        with open(os_path, mode='w') as f:
            f.write(request.form['text'])

        edudailycomment = EduDailyComment(
            edu_daily_id=id,
            commenter_id=current_user.get_id(),
            path=filename
        )
        db.session.add(edudailycomment)
        db.session.commit()

        flash('コメント投稿が完了しました。', 'success')
        return redirect(url_for('show_edu_daily', uid=edu_daily_id))

    return render_template('new_com.html', edu_daily_id=edu_daily_id)


@app.route('/edudaily/<int:id>/edit', methods=['GET'])
@login_required
def edit(id):
    edudaily = EduDaily.query.get(id)
    if current_user.get_id() != edudaily.user_id:
        flash('記事への編集権限がありません。', 'danger')
        return redirect(url_for('index'))

    os_path = os.path.join(app.root_path, 'edu_daily', edudaily.path)
    with open(os_path) as f:
        t = f.read()
    text = t

    return render_template('edit.html', **locals())


@app.route('/edudaily/<int:id>/countup', methods=['POST'])
@login_required
def countup(id):
    edudaily = EduDaily.query.get(id)
    edudaily.star = edudaily.star + 1

    db.session.merge(edudaily)
    db.session.commit()

    edudaily = EduDaily.query.get(id)

    return jsonify({'result': 'ok', 'value': edudaily.star})


@app.route('/edudaily/<int:id>/edit_com', methods=['GET'])
@login_required
def edit_com(id):
    edudailycomment = EduDailyComment.query.get(id)
    if current_user.get_id() != edudailycomment.commenter_id:
        flash('記事への編集権限がありません。', 'danger')
        return redirect(url_for('index'))

    os_path = os.path.join(app.root_path, 'edu_daily', edudailycomment.path)
    with open(os_path) as f:
        t = f.read()
    text = t

    return render_template('edit_com.html', **locals())


@app.route('/edudaily/<int:id>/countup_com', methods=['POST'])
@login_required
def countup_com(id):
    edudailycomment = EduDailyComment.query.get(id)
    edudailycomment.star = edudailycomment.star + 1

    db.session.merge(edudailycomment)
    db.session.commit()

    edudailycomment = edudailycomment.query.get(id)

    return jsonify({'result': 'ok', 'value': edudailycomment.star})


@app.route('/edudaily/<int:id>/update', methods=['POST'])
@login_required
def update(id):
    edudaily = EduDaily.query.get(id)
    if current_user.get_id() != edudaily.user_id:
        flash('記事への更新権限がありません。', 'danger')
        return redirect(url_for('index'))

    filename = edudaily.path
    os_path = os.path.join(app.root_path, 'edu_daily', filename)

    with open(os_path, mode='w') as f:
        f.write(request.form['text'])

    edudaily.title = request.form['title'],
    edudaily.updated_at = datetime.datetime.now()

    db.session.merge(edudaily)
    db.session.commit()
    flash('本文の更新が完了しました', 'success')
    return redirect(url_for('show_edu_daily', uid=id))


@app.route('/edudaily/<int:id>/update_com', methods=['POST'])
@login_required
def update_com(id):
    edudailycomment = EduDailyComment.query.get(id)
    if current_user.get_id() != edudailycomment.commenter_id:
        flash('記事への更新権限がありません。', 'danger')
        return redirect(url_for('index'))

    filename = edudailycomment.path
    edudailyID = edudailycomment.edu_daily_id
    os_path = os.path.join(app.root_path, 'edu_daily', filename)

    with open(os_path, mode='w') as f:
        f.write(request.form['text'])

    edudailycomment.updated_at = datetime.datetime.now()

    db.session.merge(edudailycomment)
    db.session.commit()
    flash('コメントの更新が完了しました', 'success')
    return redirect(url_for('show_edu_daily', uid=edudailyID))


@app.route('/edudaily/<int:id>/delete', methods=['POST'])
@login_required
def delete(id):
    edudaily = EduDaily.query.get(id)
    if current_user.get_id() != edudaily.user_id:
        flash('記事への削除権限がありません。', 'danger')
        return redirect(url_for('index'))

    edudailyfile = edudaily.path
    edudailycomment = EduDailyComment.query.filter_by(edu_daily_id=id).all()
    db.session.delete(edudaily)
    db.session.commit()

    os_path = os.path.join(app.root_path, 'edu_daily', edudailyfile)
    os.remove(os_path)

    for edudailycommentfile in edudailycomment:
        os_path = os.path.join(app.root_path, 'edu_daily',
                               edudailycommentfile.path)
        os.remove(os_path)

    flash('本文の削除が完了しました。', 'success')
    return redirect(url_for('index'))


@app.route('/edudaily/<int:id>/delete_com', methods=['POST'])
@login_required
def delete_com(id):
    edudailycomment = EduDailyComment.query.get(id)
    if current_user.get_id() != edudailycomment.commenter_id:
        flash('記事への削除権限がありません。', 'danger')
        return redirect(url_for('index'))

    filename = edudailycomment.path
    edu_daily_id = edudailycomment.edu_daily_id

    db.session.delete(edudailycomment)
    db.session.commit()

    os_path = os.path.join(app.root_path, 'edu_daily', filename)
    os.remove(os_path)

    flash('コメントの削除が完了しました。', 'success')

    return redirect(url_for('show_edu_daily', uid=edu_daily_id))


@app.route('/user/')
@login_required
def user():
    current_user_edudaily = EduDaily.query.order_by(
        EduDaily.id.desc()).filter_by(user_id=current_user.get_id()).all()
    icon = User.query.filter_by(id=current_user.get_id()).first().icon
    return render_template('user.html', **locals())


@app.route('/user/add/', methods=['GET', 'POST'])
@login_required
@admin_required
def useradd():
    if request.method == 'POST':
        username = request.form['username']
        password = generate_password_hash(request.form['password'])
        is_admin = 0
        upload_file = None
        icon_name = 'default.png'

        if User.query.filter_by(name=username).count() == 1:
            flash('ユーザ名重複しています。', 'danger')
            return redirect(request.url), 409

        if 'is_admin' in request.form:
            is_admin = 1

        if 'uploadicon' in request.files:
            upload_file = request.files['uploadicon']
            _, ext = os.path.splitext(upload_file.filename)
            icon_name = username + ext
            upload_file.save(os.path.join(app.root_path, 'static', icon_name))
        else:
            shutil.copyfile(os.path.join(app.root_path, 'static', icon_name),
                            os.path.join(
                                app.root_path, 'static', 
                                '{}.png'.format(username)))
            icon_name = '{}.png'.format(username)

        user = User(
            name=username,
            password=password,
            is_admin=is_admin,
            icon=icon_name
        )

        db.session.add(user)
        db.session.commit()
        flash('新規ユーザ追加が完了しました。', 'success')

        return redirect(url_for('userall'))

    return render_template('useradd.html')


@app.route('/user/edit/', methods=['GET', 'POST'])
@login_required
def useredit():
    if request.method == 'POST':
        username = request.form['username']
        user = User.query.filter_by(name=username).first()
        user.password = generate_password_hash(request.form['password'])

        if 'is_admin' in request.form:
            user.is_admin = 1

        if 'uploadicon' in request.files:
            upload_file = request.files['uploadicon']
            _, ext = os.path.splitext(upload_file.filename)
            upload_file.save(os.path.join(
                app.root_path, 'static', username + ext))
            user.icon = username + ext

        db.session.merge(user)
        db.session.commit()
        flash('ユーザ編集が完了しました。', 'success')

        return redirect(url_for('user'))

    return render_template('useredit.html')


@app.route('/user/all/')
@login_required
@admin_required
def userall():
    users = User.query.all()
    return render_template('userall.html', **locals())


@app.route('/user/admin/edit/<int:id>', methods=['GET', 'POST'])
@login_required
@admin_required
def useredit_admin(id):
    user = User.query.get(id)
    if request.method == 'POST':
        user = User.query.get(id)
        user.password = generate_password_hash(request.form['password'])

        if 'is_admin' in request.form:
            user.is_admin = 1
        else:
            user.is_admin = 0

        if 'uploadicon' in request.files:
            upload_file = request.files['uploadicon']
            _, ext = os.path.splitext(upload_file.filename)
            upload_file.save(os.path.join(
                app.root_path, 'static', user.name + ext))
            user.icon = user.name + ext

        user.updated_at = datetime.datetime.now()

        db.session.merge(user)
        db.session.commit()
        flash('ユーザ編集が完了しました。', 'success')

        return redirect(url_for('userall'))

    return render_template('useredit_admin.html', **locals())


@app.route('/user/admin/del/<string:username>', methods=['POST'])
@login_required
@admin_required
def userdel_admin(username):
    user = User.query.filter_by(name=username).first()
    user.is_deleted = 1
    user.updated_at = datetime.datetime.now()

    db.session.merge(user)
    db.session.commit()

    flash('ユーザを削除しました。', 'success')
    return redirect(url_for('userall'))


@app.route('/user/admin/back/<string:username>', methods=['POST'])
@login_required
@admin_required
def userback_admin(username):
    user = User.query.filter_by(name=username).first()
    user.is_deleted = 0
    user.updated_at = datetime.datetime.now()

    db.session.merge(user)
    db.session.commit()

    flash('ユーザを戻しました。', 'success')
    return redirect(url_for('userall'))

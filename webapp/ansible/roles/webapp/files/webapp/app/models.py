import datetime
from flask_login import UserMixin
from werkzeug.security import generate_password_hash, check_password_hash
from app import db


class User(UserMixin, db.Model):
    __tablename__ = 'users'
    id = db.Column(db.Integer, primary_key=True, autoincrement=True)
    name = db.Column(db.String(40), nullable=False)
    password = db.Column(db.String(120), nullable=False)
    is_admin = db.Column(db.Integer, nullable=False)
    icon = db.Column(db.String(45), nullable=False)
    created_at = db.Column(db.DateTime, nullable=False)
    updated_at = db.Column(db.DateTime, nullable=False)
    is_deleted = db.Column(db.Integer, nullable=False)

    edudaily = db.relation("EduDaily", backref="user")
    edudailycomment = db.relation("EduDailyComment", backref="user")

    def __init__(self, name=None, password=None, is_admin=0, icon=None,
                 is_deleted=0):
        self.name = name
        self.password = password
        self.is_admin = is_admin
        self.icon = icon
        now = datetime.datetime.now()
        self.created_at = now
        self.updated_at = now
        self.is_deleted = is_deleted

    def get_id(self):
        return self.id

    def get_name(self):
        return self.name

    def get_password(self):
        return self.password

    def get_admin(self):
        return self.is_admin == 1

    def get_created_at(self):
        return str(self.created_at)

    def get_updated_at(self):
        return str(self.updated_at)

    def get_all_posts(self):
        return EduDaily.query.order_by(EduDaily.updated_at.desc())

    def get_is_deleted(self):
        return self.is_deleted == 1

    def set_password(self, password):
        self.password = generate_password_hash(password)

    def check_password(self, password):
        return check_password_hash(self.password, password)


class EduDaily(db.Model):
    __tablename__ = 'edu_daily'
    id = db.Column(db.Integer, primary_key=True, autoincrement=True)
    user_id = db.Column(db.Integer, db.ForeignKey(User.id), nullable=False)
    title = db.Column(db.String(256), nullable=False)
    path = db.Column(db.String(256), nullable=False)
    star = db.Column(db.Integer, nullable=False)
    created_at = db.Column(db.DateTime, nullable=False)
    updated_at = db.Column(db.DateTime, nullable=False)

    edudailycomment = db.relation("EduDailyComment", backref="edudaily",
                                  cascade="all, delete-orphan")

    def __init__(self, user_id=None, title=None, path=None, star=0):
        self.user_id = user_id
        self.title = title
        self.path = path
        self.star = star
        now = datetime.datetime.now()
        self.created_at = now
        self.updated_at = now


class EduDailyComment(db.Model):
    __tablename__ = 'edu_daily_comment'
    id = db.Column(db.Integer, primary_key=True, autoincrement=True)
    edu_daily_id = db.Column(
        db.Integer, db.ForeignKey(EduDaily.id), nullable=False)
    commenter_id = db.Column(
        db.Integer, db.ForeignKey(User.id), nullable=False)
    path = db.Column(db.String(256), nullable=False)
    star = db.Column(db.Integer, nullable=False)
    created_at = db.Column(db.DateTime, nullable=False)
    updated_at = db.Column(db.DateTime, nullable=False)

    def __init__(self, edu_daily_id=None, commenter_id=None,
                 path=None, star=0):
        self.edu_daily_id = edu_daily_id
        self.commenter_id = commenter_id
        self.path = path
        self.star = star
        now = datetime.datetime.now()
        self.created_at = now
        self.updated_at = now

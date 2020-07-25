import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import '../application.dart';

Drawer drawer(context) {
  return Drawer(
    child: ListView(
      physics: NeverScrollableScrollPhysics(),
      padding: EdgeInsets.zero,
      children: <Widget>[
        ListTile(
          leading: Icon(Icons.home),
          title: Text(applicationName),
          onTap: () => Navigator.popUntil(context, ModalRoute.withName('/')),
        ),
        Divider(thickness: 2),
        ListTile(
          leading: Icon(Icons.list),
          title: Text('Browse APIs'),
          onTap: () {},
        ),
        ListTile(
          leading: Icon(Icons.person),
          title: Text('My APIs'),
          onTap: () {},
        ),
        ListTile(
          leading: Icon(Icons.bookmark),
          title: Text('Saved APIs'),
          onTap: () {},
        ),
        Center(
          child: Wrap(
            children: [
              FlatButton(
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(20.0),
                  side: BorderSide(),
                ),
                color: Colors.white,
                // padding: EdgeInsets.fromLTRB(20, 0, 25, 0),
                onPressed: () {},
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.add),
                    Text(
                      "Add API",
                      style: TextStyle(
                        fontSize: 14.0,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        Divider(thickness: 2),
        ListTile(
          leading: Icon(Icons.school),
          title: Text('API Design Process'),
          onTap: () {},
        ),
        ListTile(
          leading: Icon(Icons.school),
          title: Text('API Style Guide'),
          onTap: () {
            launch("https://aip.dev");
          },
        ),
      ],
    ),
  );
}